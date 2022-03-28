package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// Set a max receive message size: 500mb
const maxMsgSize = 512 * 1024 * 1024

func NewAuthConn(ctx context.Context, host string, saPath string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	var idTokenSource oauth2.TokenSource
	var err error
	audience := "https://" + strings.Split(host, ":")[0]
	if saPath == "" {
		idTokenSource, err = idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			if err.Error() != `idtoken: credential must be service_account, found "authorized_user"` {
				return nil, fmt.Errorf("idtoken.NewTokenSource: %v", err)
			}
			gts, err := google.DefaultTokenSource(ctx)
			if err != nil {
				return nil, fmt.Errorf("attempt to use Application Default Credentials failed: %v", err)
			}
			idTokenSource = gts
		}
	} else {
		idTokenSource, err = idtoken.NewTokenSource(ctx, audience, idtoken.WithCredentialsFile(saPath))
		if err != nil {
			return nil, fmt.Errorf("unable to create TokenSource: %v", err)
		}
	}

	opts = append(opts, grpc.WithAuthority(host))

	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})

	opts = append(opts,
		[]grpc.DialOption{
			grpc.WithTransportCredentials(cred),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxMsgSize)),
			grpc.WithPerRPCCredentials(oauth.TokenSource{idTokenSource}),
		}...,
	)

	return grpc.Dial(host, opts...)
}

func InsecureConn(host string) (*grpc.ClientConn, error) {
	return grpc.Dial(host,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxMsgSize)),
	)
}
