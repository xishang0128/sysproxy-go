# sysproxy-go

一个用于设置系统代理的工具

支持 windows/linux/macos

windows 使用 win32 api 设置代理，支持拨号（需要英文接口名称）

linux 使用 kwriteconfig5(6)/gsettings 设置代理，仅支持 kde/gnome

macOS 使用 networksetup 为所有接口设置代理，或者仅为使用中的接口设置
