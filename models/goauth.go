package models

import "golang.org/x/oauth2"

type Goauth struct {
	Token  *oauth2.Token
	Client *oauth2.Config
}
