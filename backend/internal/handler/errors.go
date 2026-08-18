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
	CodeBadRequest = 100001
	CodeInvalidID  = 100002
	CodeInternal   = 100003
)

const (
	CodeGenreListFailed    = 300001
	CodePlatformListFailed = 300002
	CodeLLMConfigFailed    = 300003
	CodeGenreCreateFailed  = 300004
	CodeGenreUpdateFailed  = 300005
	CodeGenreDeleteFailed  = 300006
)

const (
	CodeStatsFailed          = 400001
	CodePlatformCreateFailed = 400007
	CodePlatformUpdateFailed = 400008
	CodePlatformDeleteFailed = 400009
	CodeLLMCreateFailed      = 400010
	CodeLLMUpdateFailed      = 400011
	CodeLLMDeleteFailed      = 400012
	CodeLLMDefaultFailed     = 400013
	CodeLLMUsageFailed       = 400014
)

const (
	CodeIncompleteBook      = 500001
	CodeBookCreateFailed    = 500002
	CodeBookListFailed      = 500003
	CodeBookBusy            = 500004
	CodeBookInitFailed      = 500005
	CodeBookWriteFailed     = 500006
	CodeBookDeleteFailed    = 500007
	CodeChapterDeleteFailed = 500008
)

var (
	ErrBadRequest = APIError{http.StatusBadRequest, CodeBadRequest, "参数错误"}
	ErrInvalidID  = APIError{http.StatusBadRequest, CodeInvalidID, "无效的ID"}
	ErrInternal   = APIError{http.StatusInternalServerError, CodeInternal, "服务器内部错误"}
)

var (
	ErrGenreListFailed    = APIError{http.StatusInternalServerError, CodeGenreListFailed, "获取题材列表失败"}
	ErrPlatformListFailed = APIError{http.StatusInternalServerError, CodePlatformListFailed, "获取平台列表失败"}
	ErrLLMConfigFailed    = APIError{http.StatusInternalServerError, CodeLLMConfigFailed, "获取模型配置失败"}
	ErrGenreCreateFailed  = APIError{http.StatusInternalServerError, CodeGenreCreateFailed, "创建题材失败"}
	ErrGenreUpdateFailed  = APIError{http.StatusInternalServerError, CodeGenreUpdateFailed, "更新题材失败"}
	ErrGenreDeleteFailed  = APIError{http.StatusInternalServerError, CodeGenreDeleteFailed, "删除题材失败"}
)

var (
	ErrStatsFailed          = APIError{http.StatusInternalServerError, CodeStatsFailed, "获取统计数据失败"}
	ErrPlatformCreateFailed = APIError{http.StatusInternalServerError, CodePlatformCreateFailed, "创建平台失败"}
	ErrPlatformUpdateFailed = APIError{http.StatusInternalServerError, CodePlatformUpdateFailed, "更新平台失败"}
	ErrPlatformDeleteFailed = APIError{http.StatusInternalServerError, CodePlatformDeleteFailed, "删除平台失败"}
	ErrLLMCreateFailed      = APIError{http.StatusInternalServerError, CodeLLMCreateFailed, "创建模型配置失败"}
	ErrLLMUpdateFailed      = APIError{http.StatusInternalServerError, CodeLLMUpdateFailed, "更新模型配置失败"}
	ErrLLMDeleteFailed      = APIError{http.StatusInternalServerError, CodeLLMDeleteFailed, "删除模型配置失败"}
	ErrLLMDefaultFailed     = APIError{http.StatusInternalServerError, CodeLLMDefaultFailed, "设置默认模型失败"}
	ErrLLMUsageFailed       = APIError{http.StatusInternalServerError, CodeLLMUsageFailed, "获取Token用量失败"}
)

var (
	ErrIncompleteBook      = APIError{http.StatusBadRequest, CodeIncompleteBook, "请填写完整的书籍信息"}
	ErrBookCreateFailed    = APIError{http.StatusInternalServerError, CodeBookCreateFailed, "创建书籍失败"}
	ErrBookListFailed      = APIError{http.StatusInternalServerError, CodeBookListFailed, "获取书籍列表失败"}
	ErrBookBusy            = APIError{http.StatusConflict, CodeBookBusy, "该书正在处理中，请稍后再试"}
	ErrBookInitFailed      = APIError{http.StatusInternalServerError, CodeBookInitFailed, "初始化书籍失败"}
	ErrBookWriteFailed     = APIError{http.StatusInternalServerError, CodeBookWriteFailed, "写作失败"}
	ErrBookDeleteFailed    = APIError{http.StatusInternalServerError, CodeBookDeleteFailed, "删除书籍失败"}
	ErrChapterDeleteFailed = APIError{http.StatusInternalServerError, CodeChapterDeleteFailed, "删除章节失败"}
)
