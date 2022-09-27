package authn

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
)

const (
	TeleportUID = "0"
)

type StaticTokenAuthenticator struct {
	tokens map[string]*user.DefaultInfo
}

// New returns a StaticTokenAuthenticator for a single token
func New(tokens map[string]*user.DefaultInfo) *StaticTokenAuthenticator {
	return &StaticTokenAuthenticator{
		tokens: tokens,
	}
}

// NewCSV returns a StaticTokenAuthenticator, populated from a CSV file.
// The CSV file must contain records in the format "token,username,useruid"
func NewCSV(path string) (*StaticTokenAuthenticator, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	recordNum := 0
	tokens := make(map[string]*user.DefaultInfo)
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 3 {
			return nil, fmt.Errorf("token file '%s' must have at least 3 columns (token, user name, user uid), found %d", path, len(record))
		}

		recordNum++
		if record[0] == "" {
			klog.Warningf("empty token has been found in token file '%s', record number '%d'", path, recordNum)
			continue
		}

		obj := &user.DefaultInfo{
			Name: record[1],
			UID:  record[2],
		}
		if _, exist := tokens[record[0]]; exist {
			klog.Warningf("duplicate token has been found in token file '%s', record number '%d'", path, recordNum)
		}
		tokens[record[0]] = obj

		if len(record) >= 4 {
			obj.Groups = strings.Split(record[3], ",")
		}
	}

	return &StaticTokenAuthenticator{
		tokens: tokens,
	}, nil
}

func (a *StaticTokenAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	user, ok := a.tokens[value]
	if !ok {
		return nil, false, nil
	}
	return &authenticator.Response{User: user}, true, nil
}

func (a *StaticTokenAuthenticator) GetTokens() map[string]*user.DefaultInfo {
	return a.tokens
}

func (a *StaticTokenAuthenticator) GetToken(uid string) string {
	for token, userInfo := range a.tokens {
		if userInfo.UID == uid {
			return token
		}
	}
	return ""
}
