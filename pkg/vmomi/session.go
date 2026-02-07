package vmomi

import (
	"context"
	"errors"

	"github.com/vmware/govmomi/vim25"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
	sx "github.com/9506hqwy/vmomi-event-source/pkg/vmomi/sessionex"
)

type ConnInfo struct {
	URL         string
	User        string
	Password    string
	NoVerifySSL bool
}

func login(ctx context.Context) (*vim25.Client, error) {
	info, err := GetTarget(ctx)
	if err != nil {
		return nil, err
	}

	c, err := sx.Login(ctx, info.URL, info.User, info.Password, info.NoVerifySSL)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func GetTarget(ctx context.Context) (i *ConnInfo, err error) {
	url, ok := ctx.Value(flag.TargetURLKey{}).(string)
	if !ok {
		return nil, errors.New("target_url not found in context")
	}

	user, ok := ctx.Value(flag.TargetUserKey{}).(string)
	if !ok {
		return nil, errors.New("target_user not found in context")
	}

	password, ok := ctx.Value(flag.TargetPasswordKey{}).(string)
	if !ok {
		return nil, errors.New("target_password not found in context")
	}

	noVerifySSL, ok := ctx.Value(flag.TargetNoVerifySSLKey{}).(bool)
	if !ok {
		return nil, errors.New("target_no_verify_ssl not found in context")
	}

	c := ConnInfo{
		URL:         url,
		User:        user,
		Password:    password,
		NoVerifySSL: noVerifySSL,
	}

	return &c, nil
}
