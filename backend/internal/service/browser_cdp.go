package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type cdpTarget struct {
	ID                   string `json:"id"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type BrowserSessionStatus struct {
	Configured           bool   `json:"configured"`
	Connected            bool   `json:"connected"`
	FanqieAccessible     bool   `json:"fanqie_accessible"`
	RequiresVerification bool   `json:"requires_verification"`
	RequiresLogin        bool   `json:"requires_login"`
	HasCookies           bool   `json:"has_cookies"`
	Ready                bool   `json:"ready"`
	Message              string `json:"message"`
	CDPURL               string `json:"cdp_url"`
	CookieHeader         string `json:"-"`
}

type browserLaunchOptions struct {
	AutoLaunch    bool
	ChromeAppName string
	UserDataDir   string
}

type cdpClient struct {
	conn   net.Conn
	reader *bufio.Reader
	nextID int
}

const fanqieSessionCheckURL = "https://fanqienovel.com/reader/7582240916874740286"

func checkFanqieBrowserSession(ctx context.Context, cdpURL string, timeout time.Duration, launch browserLaunchOptions) *BrowserSessionStatus {
	status := &BrowserSessionStatus{
		Configured: strings.TrimSpace(cdpURL) != "",
		CDPURL:     strings.TrimSpace(cdpURL),
	}
	if !status.Configured {
		status.Message = "未配置本地 Chrome DevTools 地址"
		return status
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseURL := strings.TrimRight(cdpURL, "/")
	if err := pingCDP(ctx, baseURL); err != nil {
		if launchErr := ensureChromeDevTools(ctx, cdpURL, launch); launchErr != nil {
			status.Message = "无法自动启动本地 Chrome DevTools：" + launchErr.Error()
			return status
		}
		if err := pingCDP(ctx, baseURL); err != nil {
			status.Message = "无法连接本地 Chrome DevTools，请确认 Chrome 已启动"
			return status
		}
	}
	status.Connected = true

	target, err := createBackgroundCDPTarget(ctx, baseURL, fanqieSessionCheckURL)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	defer closeCDPTarget(context.Background(), baseURL, target.ID)

	client, err := dialCDP(ctx, target.WebSocketDebuggerURL)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	defer client.Close()

	_, _ = client.Call(ctx, "Page.enable", nil)
	_, _ = client.Call(ctx, "Runtime.enable", nil)
	_, _ = client.Call(ctx, "Page.navigate", map[string]any{"url": fanqieSessionCheckURL})
	time.Sleep(2 * time.Second)

	result, err := client.Evaluate(ctx, fanqieReaderExtractScript)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	body := result.Body
	status.RequiresVerification = strings.Contains(body, "验证码") || strings.Contains(body, "TTGCaptcha") || strings.Contains(body, "验证")
	status.RequiresLogin = strings.Contains(body, "登录")
	status.FanqieAccessible = !status.RequiresVerification && strings.TrimSpace(body) != "" && isUsableChapterContent(result.Content)
	cookieHeader, _ := client.CookieHeader(ctx)
	status.CookieHeader = cookieHeader
	status.HasCookies = hasLikelyFanqieLoginCookie(cookieHeader)
	status.Ready = status.Connected && status.FanqieAccessible && !status.RequiresVerification && !status.RequiresLogin && status.HasCookies
	if status.Ready {
		status.Message = "本地浏览器会话可用"
	} else if status.RequiresVerification {
		status.Message = "本地 Chrome 仍需要完成番茄验证码或登录验证"
	} else if status.RequiresLogin || !status.HasCookies {
		status.Message = "本地 Chrome 尚未登录番茄，请先登录"
	} else {
		status.Message = "本地 Chrome 未能读取到真实章节正文，请确认登录后 reader 页面可阅读全文"
	}
	return status
}

func openFanqieBrowserPage(ctx context.Context, cdpURL string, launch browserLaunchOptions) *BrowserSessionStatus {
	status := &BrowserSessionStatus{
		Configured: strings.TrimSpace(cdpURL) != "",
		CDPURL:     strings.TrimSpace(cdpURL),
	}
	if !status.Configured {
		status.Message = "未配置本地 Chrome DevTools 地址"
		return status
	}
	baseURL := strings.TrimRight(cdpURL, "/")
	if err := pingCDP(ctx, baseURL); err != nil {
		if launchErr := ensureChromeDevTools(ctx, cdpURL, launch); launchErr != nil {
			status.Message = "无法自动启动本地 Chrome DevTools：" + launchErr.Error()
			return status
		}
		if err := pingCDP(ctx, baseURL); err != nil {
			status.Message = "无法连接本地 Chrome DevTools，请确认 Chrome 已启动"
			return status
		}
	}
	status.Connected = true
	if _, err := createCDPTarget(ctx, baseURL, fanqieSessionCheckURL); err != nil {
		status.Message = err.Error()
		return status
	}
	status.Message = "已在本地 Chrome 打开番茄 reader 页面，请完成登录/验证码并确认能看到正文后重新扫描"
	return status
}

func ensureChromeDevTools(ctx context.Context, cdpURL string, launch browserLaunchOptions) error {
	if !launch.AutoLaunch {
		return fmt.Errorf("自动启动未开启")
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("当前只支持 macOS 自动启动 Chrome")
	}
	port := cdpPort(cdpURL)
	appName := strings.TrimSpace(launch.ChromeAppName)
	if appName == "" {
		appName = "Google Chrome"
	}
	userDataDir := strings.TrimSpace(launch.UserDataDir)
	if userDataDir == "" {
		userDataDir = os.ExpandEnv("$HOME/.whwriter-chrome")
	}
	userDataDir = os.ExpandEnv(userDataDir)
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		return fmt.Errorf("创建 Chrome 用户目录失败：%w", err)
	}
	cmd := exec.CommandContext(ctx, "open", "-na", appName, "--args",
		"--remote-debugging-port="+port,
		"--user-data-dir="+userDataDir,
		"--no-first-run",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动 Chrome 失败：%w", err)
	}
	baseURL := strings.TrimRight(cdpURL, "/")
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if err := pingCDP(ctx, baseURL); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("Chrome 已启动但 DevTools 端口未就绪")
}

func cdpPort(cdpURL string) string {
	u, err := url.Parse(cdpURL)
	if err == nil && u.Port() != "" {
		return u.Port()
	}
	return "9222"
}

func pingCDP(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/json/version", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Chrome DevTools 状态异常：%s", resp.Status)
	}
	return nil
}

func fetchFanqieChaptersWithBrowser(ctx context.Context, cdpURL string, chapterIDs []string, timeout time.Duration, launch browserLaunchOptions) ([]fanqieChapter, error) {
	if strings.TrimSpace(cdpURL) == "" {
		return nil, fmt.Errorf("未配置本地浏览器 DevTools 地址")
	}
	if err := ensureChromeDevTools(ctx, cdpURL, launch); err != nil {
		if pingErr := pingCDP(ctx, strings.TrimRight(cdpURL, "/")); pingErr != nil {
			return nil, err
		}
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	baseURL := strings.TrimRight(cdpURL, "/")
	target, err := createBackgroundCDPTarget(ctx, baseURL, "about:blank")
	if err != nil {
		return nil, err
	}
	defer closeCDPTarget(context.Background(), baseURL, target.ID)

	client, err := dialCDP(ctx, target.WebSocketDebuggerURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if _, err := client.Call(ctx, "Page.enable", nil); err != nil {
		return nil, err
	}
	if _, err := client.Call(ctx, "Runtime.enable", nil); err != nil {
		return nil, err
	}

	chapters := make([]fanqieChapter, 0, len(chapterIDs))
	for i, chapterID := range chapterIDs {
		chapter, err := fetchFanqieChapterWithBrowserClient(ctx, client, chapterID, i+1, timeout)
		if err != nil {
			return chapters, err
		}
		if strings.TrimSpace(chapter.Content) != "" {
			chapters = append(chapters, *chapter)
		}
	}
	return chapters, nil
}

func fetchFanqieChapterWithBrowser(parent context.Context, cdpURL, chapterID string, no int, timeout time.Duration) (*fanqieChapter, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	baseURL := strings.TrimRight(cdpURL, "/")
	target, err := createBackgroundCDPTarget(ctx, baseURL, "about:blank")
	if err != nil {
		return nil, err
	}
	defer closeCDPTarget(context.Background(), baseURL, target.ID)

	client, err := dialCDP(ctx, target.WebSocketDebuggerURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if _, err := client.Call(ctx, "Page.enable", nil); err != nil {
		return nil, err
	}
	if _, err := client.Call(ctx, "Runtime.enable", nil); err != nil {
		return nil, err
	}
	return fetchFanqieChapterWithBrowserClient(ctx, client, chapterID, no, timeout)
}

func fetchFanqieChapterWithBrowserClient(parent context.Context, client *cdpClient, chapterID string, no int, timeout time.Duration) (*fanqieChapter, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	readerURL := "https://fanqienovel.com/reader/" + strings.TrimSpace(chapterID)
	if _, err := client.Call(ctx, "Page.navigate", map[string]any{"url": readerURL}); err != nil {
		return nil, err
	}

	var lastBody string
	for {
		select {
		case <-ctx.Done():
			if strings.Contains(lastBody, "验证码") || strings.Contains(lastBody, "TTGCaptcha") {
				return nil, fmt.Errorf("本地浏览器仍停留在番茄验证码页，请在 Chrome 中完成验证后重试")
			}
			return nil, fmt.Errorf("本地浏览器抓取章节超时")
		default:
		}

		result, err := client.Evaluate(ctx, fanqieReaderExtractScript)
		if err == nil && isUsableChapterContent(result.Content) {
			return &fanqieChapter{
				No:      no,
				Title:   result.Title,
				Content: normalizeSampleText(result.Content),
			}, nil
		}
		lastBody = result.Body
		time.Sleep(2 * time.Second)
	}
}

func createBackgroundCDPTarget(ctx context.Context, baseURL, targetURL string) (*cdpTarget, error) {
	browserWS, err := getBrowserWebSocketURL(ctx, baseURL)
	if err != nil {
		return nil, err
	}
	client, err := dialCDP(ctx, browserWS)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	result, err := client.Call(ctx, "Target.createTarget", map[string]any{
		"url":        targetURL,
		"background": true,
	})
	if err != nil {
		return nil, err
	}
	var parsed struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, err
	}
	if parsed.TargetID == "" {
		return nil, fmt.Errorf("Chrome DevTools 未返回后台页面 targetId")
	}
	target, err := getCDPTarget(ctx, baseURL, parsed.TargetID)
	if err != nil {
		_, _ = client.Call(context.Background(), "Target.closeTarget", map[string]any{"targetId": parsed.TargetID})
		return nil, err
	}
	return target, nil
}

func getBrowserWebSocketURL(ctx context.Context, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/json/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Chrome DevTools 状态异常：%s", resp.Status)
	}
	var parsed struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("Chrome DevTools 未返回 browser WebSocket 地址")
	}
	return parsed.WebSocketDebuggerURL, nil
}

func getCDPTarget(ctx context.Context, baseURL, targetID string) (*cdpTarget, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/json/list", nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		var targets []cdpTarget
		decodeErr := json.NewDecoder(resp.Body).Decode(&targets)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return nil, decodeErr
		}
		for _, target := range targets {
			if target.ID == targetID && target.WebSocketDebuggerURL != "" {
				return &target, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("未找到 Chrome 后台页面 target：%s", targetID)
}

func createCDPTarget(ctx context.Context, baseURL, targetURL string) (*cdpTarget, error) {
	endpoint := baseURL + "/json/new?" + url.QueryEscape(targetURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接本地 Chrome DevTools 失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return createCDPTargetWithGet(ctx, endpoint)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("创建 Chrome 标签页失败：%s", resp.Status)
	}
	var target cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, err
	}
	if target.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("Chrome DevTools 未返回页面 WebSocket 地址")
	}
	return &target, nil
}

func createCDPTargetWithGet(ctx context.Context, endpoint string) (*cdpTarget, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接本地 Chrome DevTools 失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("创建 Chrome 标签页失败：%s", resp.Status)
	}
	var target cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, err
	}
	if target.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("Chrome DevTools 未返回页面 WebSocket 地址")
	}
	return &target, nil
}

func closeCDPTarget(ctx context.Context, baseURL, targetID string) {
	if strings.TrimSpace(targetID) == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/json/close/"+url.PathEscape(targetID), nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func dialCDP(ctx context.Context, rawURL string) (*cdpClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ws" {
		return nil, fmt.Errorf("仅支持本地 ws DevTools 地址：%s", rawURL)
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("连接 Chrome 页面 WebSocket 失败：%w", err)
	}
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		_ = conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}
	request := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, u.Host, key)
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return nil, fmt.Errorf("Chrome WebSocket 握手失败：%s", resp.Status)
	}
	if !validWebSocketAccept(key, resp.Header.Get("Sec-WebSocket-Accept")) {
		_ = conn.Close()
		return nil, fmt.Errorf("Chrome WebSocket 握手校验失败")
	}
	return &cdpClient{conn: conn, reader: reader}, nil
}

func validWebSocketAccept(key, got string) bool {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	want := base64.StdEncoding.EncodeToString(sum[:])
	return strings.TrimSpace(got) == want
}

func (c *cdpClient) Close() error {
	return c.conn.Close()
}

func (c *cdpClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.nextID++
	id := c.nextID
	payload := map[string]any{"id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := c.writeTextFrame(data); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		message, err := c.readMessage()
		if err != nil {
			return nil, err
		}
		var resp struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("%s", resp.Error.Message)
		}
		return resp.Result, nil
	}
}

type readerExtractResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Body    string `json:"body"`
}

func (c *cdpClient) Evaluate(ctx context.Context, script string) (*readerExtractResult, error) {
	result, err := c.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    script,
		"returnByValue": true,
	})
	if err != nil {
		return &readerExtractResult{}, err
	}
	var parsed struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return &readerExtractResult{}, err
	}
	var out readerExtractResult
	if err := json.Unmarshal([]byte(parsed.Result.Value), &out); err != nil {
		return &readerExtractResult{}, err
	}
	out.Title = html.UnescapeString(strings.TrimSpace(out.Title))
	out.Content = html.UnescapeString(strings.TrimSpace(out.Content))
	out.Body = html.UnescapeString(strings.TrimSpace(out.Body))
	return &out, nil
}

func (c *cdpClient) CookieHeader(ctx context.Context) (string, error) {
	result, err := c.Call(ctx, "Network.getAllCookies", nil)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Cookies []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Domain string `json:"domain"`
		} `json:"cookies"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return "", err
	}
	pairs := make([]string, 0, len(parsed.Cookies))
	seen := make(map[string]struct{}, len(parsed.Cookies))
	for _, cookie := range parsed.Cookies {
		if !strings.Contains(cookie.Domain, "fanqienovel.com") {
			continue
		}
		if cookie.Name == "" {
			continue
		}
		if _, ok := seen[cookie.Name]; ok {
			continue
		}
		seen[cookie.Name] = struct{}{}
		pairs = append(pairs, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(pairs, "; "), nil
}

func hasLikelyFanqieLoginCookie(cookieHeader string) bool {
	for _, name := range []string{"sessionid", "sid_tt", "uid_tt", "uid_tt_ss", "sid_guard"} {
		if strings.Contains(cookieHeader, name+"=") {
			return true
		}
	}
	return false
}

func (c *cdpClient) writeTextFrame(payload []byte) error {
	var header bytes.Buffer
	header.WriteByte(0x81)
	length := len(payload)
	switch {
	case length < 126:
		header.WriteByte(byte(0x80 | length))
	case length <= 65535:
		header.WriteByte(0x80 | 126)
		_ = binary.Write(&header, binary.BigEndian, uint16(length))
	default:
		header.WriteByte(0x80 | 127)
		_ = binary.Write(&header, binary.BigEndian, uint64(length))
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header.Write(mask)
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%4]
	}
	if _, err := c.conn.Write(header.Bytes()); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

func (c *cdpClient) readMessage() ([]byte, error) {
	var payload bytes.Buffer
	for {
		frame, fin, opcode, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x1, 0x0:
			payload.Write(frame)
			if fin {
				return payload.Bytes(), nil
			}
		case 0x8:
			return nil, io.EOF
		case 0x9:
			_ = c.writeControlFrame(0xA, frame)
		}
	}
}

func (c *cdpClient) readFrame() ([]byte, bool, byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return nil, false, 0, err
	}
	fin := header[0]&0x80 != 0
	opcode := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7f)
	switch length {
	case 126:
		var n uint16
		if err := binary.Read(c.reader, binary.BigEndian, &n); err != nil {
			return nil, false, 0, err
		}
		length = uint64(n)
	case 127:
		if err := binary.Read(c.reader, binary.BigEndian, &length); err != nil {
			return nil, false, 0, err
		}
	}
	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.reader, mask); err != nil {
			return nil, false, 0, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, false, 0, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, fin, opcode, nil
}

func (c *cdpClient) writeControlFrame(opcode byte, payload []byte) error {
	if len(payload) > 125 {
		payload = payload[:125]
	}
	var header bytes.Buffer
	header.WriteByte(0x80 | opcode)
	header.WriteByte(byte(0x80 | len(payload)))
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header.Write(mask)
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%4]
	}
	if _, err := c.conn.Write(header.Bytes()); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

const fanqieReaderExtractScript = `JSON.stringify((() => {
  const pick = (selectors) => {
    for (const selector of selectors) {
      const el = document.querySelector(selector);
      const text = el?.innerText || el?.textContent || "";
      if (text.trim()) return text.trim();
    }
    return "";
  };
  const state = window.__INITIAL_STATE__ || {};
  const chapterData = state.reader?.chapterData || state.reader?.chapterData?.chapterData || {};
  const title = pick(["h1", ".reader-title", ".muye-reader-title"]) || chapterData.title || document.title || "";
  const content = pick([".muye-reader-content", ".reader-content"]) || chapterData.content || "";
  const body = (document.body?.innerText || "").slice(0, 800);
  return { title, content, body };
})())`
