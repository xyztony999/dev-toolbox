package clientsettings

import "errors"

// ErrBusy 表示设置文件被其他进程锁定。
var ErrBusy = errors.New("CLIENT_SETTINGS_BUSY")
