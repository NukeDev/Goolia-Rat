package utils

import (
	"github.com/blackfireio/osinfo"
)

func GetOsInfo() (*osinfo.OSInfo, error) {
	info, err := osinfo.GetOSInfo()
	if err != nil {
		return nil, err
	}
	return info, nil
}
