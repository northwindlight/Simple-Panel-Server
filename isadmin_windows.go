//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

func isAdmin() {
	var sid *windows.SID
	_ = windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)

	admin, _ := windows.Token(0).IsMember(sid)
	if !admin {
		logrus.Error("This application requires ADMINISTRATOR privileges.")
		logrus.Error("Please right-click and select 'Run as administrator'")
		logrus.Error("")
		logrus.Error("Press Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	} else {
		logrus.Info("Application started with administrator privileges")
	}
}
