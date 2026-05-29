package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e APIError) JSON(c *gin.Context) {
	c.JSON(e.StatusCode, ErrorResponse{Code: e.Code, Message: e.Message})
}

func (e APIError) Abort(c *gin.Context) {
	c.AbortWithStatusJSON(e.StatusCode, ErrorResponse{Code: e.Code, Message: e.Message})
}

func ErrJSON(c *gin.Context, statusCode int, code int, msg string) {
	c.JSON(statusCode, ErrorResponse{Code: code, Message: msg})
}

const (
	CodeBadRequest   = 100001
	CodeInvalidID    = 100002
	CodeInternal     = 100003
	CodeUnauthorized = 100004
)

const (
	CodeInvalidEmail          = 200001
	CodeSendCodeFailed        = 200002
	CodeIncompleteRegister    = 200003
	CodeRegisterFailed        = 200004
	CodeMissingCredentials    = 200005
	CodeLoginFailed           = 200006
	CodeAdminNoPassword       = 200007
	CodeIncompleteChangePwd   = 200008
	CodeChangePwdFailed       = 200009
	CodeWeakPassword          = 200010
	CodeInvalidCode           = 200011
	CodeEmailAlreadyExists    = 200012
	CodeUsernameAlreadyExists = 200013
	CodeInvalidCredentials    = 200014
	CodeAccountDisabled       = 200015
	CodeInvalidUsername       = 200016
)

const (
	CodeGenreListFailed    = 300001
	CodePlatformListFailed = 300002
	CodeLLMConfigFailed    = 300003
	CodeGenreCreateFailed  = 300004
	CodeGenreUpdateFailed  = 300005
	CodeGenreDeleteFailed  = 300006
	CodeGenreForbidden     = 300007
)

const (
	CodeStatsFailed          = 400001
	CodeUserListFailed       = 400002
	CodeInvalidStatus        = 400003
	CodeUpdateStatusFailed   = 400004
	CodeAmountZero           = 400005
	CodeRechargeFailed       = 400006
	CodePlatformCreateFailed = 400007
	CodePlatformUpdateFailed = 400008
	CodePlatformDeleteFailed = 400009
)

const (
	CodeIncompleteBook   = 500001
	CodeBookCreateFailed = 500002
	CodeBookListFailed   = 500003
)

var (
	ErrBadRequest   = APIError{http.StatusBadRequest, CodeBadRequest, "参数错误"}
	ErrInvalidID    = APIError{http.StatusBadRequest, CodeInvalidID, "无效的ID"}
	ErrInternal     = APIError{http.StatusInternalServerError, CodeInternal, "服务器内部错误"}
	ErrUnauthorized = APIError{http.StatusUnauthorized, CodeUnauthorized, "未登录"}
)

var (
	ErrInvalidEmail        = APIError{http.StatusBadRequest, CodeInvalidEmail, "请输入有效的邮箱地址"}
	ErrSendCodeFailed      = APIError{http.StatusInternalServerError, CodeSendCodeFailed, "发送验证码失败"}
	ErrIncompleteRegister  = APIError{http.StatusBadRequest, CodeIncompleteRegister, "请填写完整的注册信息"}
	ErrRegisterFailed      = APIError{http.StatusInternalServerError, CodeRegisterFailed, "注册失败，请稍后重试"}
	ErrMissingCredentials  = APIError{http.StatusBadRequest, CodeMissingCredentials, "请输入邮箱和密码"}
	ErrLoginFailed         = APIError{http.StatusInternalServerError, CodeLoginFailed, "登录失败，请稍后重试"}
	ErrAdminNoPassword     = APIError{http.StatusForbidden, CodeAdminNoPassword, "管理员不支持修改密码"}
	ErrIncompleteChangePwd = APIError{http.StatusBadRequest, CodeIncompleteChangePwd, "请填写完整信息"}
	ErrChangePwdFailed     = APIError{http.StatusInternalServerError, CodeChangePwdFailed, "修改密码失败"}

	ErrWeakPassword          = APIError{http.StatusBadRequest, CodeWeakPassword, "密码强度不足：需要至少8位，包含大小写字母和数字"}
	ErrInvalidCode           = APIError{http.StatusBadRequest, CodeInvalidCode, "验证码无效或已过期"}
	ErrEmailAlreadyExists    = APIError{http.StatusBadRequest, CodeEmailAlreadyExists, "邮箱已被注册"}
	ErrUsernameAlreadyExists = APIError{http.StatusBadRequest, CodeUsernameAlreadyExists, "用户名已被占用"}
	ErrInvalidCredentials    = APIError{http.StatusUnauthorized, CodeInvalidCredentials, "邮箱或密码错误"}
	ErrAccountDisabled       = APIError{http.StatusUnauthorized, CodeAccountDisabled, "账户已被禁用"}
	ErrInvalidUsername       = APIError{http.StatusBadRequest, CodeInvalidUsername, "用户名需为2-16个字符"}
)

var (
	ErrGenreListFailed    = APIError{http.StatusInternalServerError, CodeGenreListFailed, "获取题材列表失败"}
	ErrPlatformListFailed = APIError{http.StatusInternalServerError, CodePlatformListFailed, "获取平台列表失败"}
	ErrLLMConfigFailed    = APIError{http.StatusInternalServerError, CodeLLMConfigFailed, "获取模型配置失败"}
	ErrGenreCreateFailed  = APIError{http.StatusInternalServerError, CodeGenreCreateFailed, "创建题材失败"}
	ErrGenreUpdateFailed  = APIError{http.StatusInternalServerError, CodeGenreUpdateFailed, "更新题材失败"}
	ErrGenreDeleteFailed  = APIError{http.StatusInternalServerError, CodeGenreDeleteFailed, "删除题材失败"}
	ErrGenreForbidden     = APIError{http.StatusForbidden, CodeGenreForbidden, "无权操作该题材"}
)

var (
	ErrStatsFailed          = APIError{http.StatusInternalServerError, CodeStatsFailed, "获取统计数据失败"}
	ErrUserListFailed       = APIError{http.StatusInternalServerError, CodeUserListFailed, "获取用户列表失败"}
	ErrInvalidStatus        = APIError{http.StatusBadRequest, CodeInvalidStatus, "无效的状态值，仅允许 active 或 disabled"}
	ErrUpdateStatusFailed   = APIError{http.StatusInternalServerError, CodeUpdateStatusFailed, "更新状态失败"}
	ErrAmountZero           = APIError{http.StatusBadRequest, CodeAmountZero, "金额不能为0"}
	ErrRechargeFailed       = APIError{http.StatusInternalServerError, CodeRechargeFailed, "充值失败"}
	ErrPlatformCreateFailed = APIError{http.StatusInternalServerError, CodePlatformCreateFailed, "创建平台失败"}
	ErrPlatformUpdateFailed = APIError{http.StatusInternalServerError, CodePlatformUpdateFailed, "更新平台失败"}
	ErrPlatformDeleteFailed = APIError{http.StatusInternalServerError, CodePlatformDeleteFailed, "删除平台失败"}
)

var (
	ErrIncompleteBook   = APIError{http.StatusBadRequest, CodeIncompleteBook, "请填写完整的书籍信息"}
	ErrBookCreateFailed = APIError{http.StatusInternalServerError, CodeBookCreateFailed, "创建书籍失败"}
	ErrBookListFailed   = APIError{http.StatusInternalServerError, CodeBookListFailed, "获取书籍列表失败"}
)
