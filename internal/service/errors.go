package service

import "errors"

var (
	ErrNotFound      = errors.New("记录不存在")
	ErrInvalidStatus = errors.New("当前状态不允许此操作")
	ErrBadRequest    = errors.New("请求参数无效")
)
