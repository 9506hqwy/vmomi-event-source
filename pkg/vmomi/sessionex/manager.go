package sessionex

import (
	"context"
	"net/url"
	"time"

	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
)

func Login(
	ctx context.Context,
	endpoint string,
	username string,
	password string,
	noVerifySSL bool,
) (*vim25.Client, error) {
	u, err := soap.ParseURL(endpoint)
	if err != nil {
		return nil, err
	}

	sc := soap.NewClient(u, noVerifySSL)
	vc, err := ExecCallAPI(
		ctx,
		func(cctx context.Context) (*vim25.Client, error) {
			return vim25.NewClient(cctx, sc)
		},
	)
	if err != nil {
		return nil, err
	}

	sm := session.NewManager(vc)

	cred := url.UserPassword(username, password)
	_, err = ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, sm.Login(cctx, cred)
		},
	)
	if err != nil {
		return nil, err
	}

	return vc, nil
}

func Logout(ctx context.Context, c *vim25.Client) error {
	sm := session.NewManager(c)

	_, err := ExecCallAPI(
		ctx,
		func(cctx context.Context) (int, error) {
			return 0, sm.Logout(cctx)
		},
	)

	return err
}

func ExecCallAPI[T any](
	ctx context.Context,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	timeout, ok := ctx.Value(flag.TargetTimeoutKey{}).(int)
	//revive:disable:add-constant
	if !ok || timeout < 0 {
		timeout = 10
	}
	//revive:enable:add-constant

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	return fn(cctx)
}
