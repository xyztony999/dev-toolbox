package clientsettings

import "dev-toolbox/internal/userstate"

// userstateGet 读取用户级运行时登记，探测逻辑的唯一事实源。
func userstateGet() (userstate.State, error) {
	return userstate.Get()
}
