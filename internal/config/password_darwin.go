package config

import (
	"github.com/bgentry/speakeasy"
	keychain "github.com/keybase/go-keychain"
	"github.com/pkg/errors"
)

func loadJIRAPassword(url, username string) (string, error) {
	var err error
	item := keychain.NewItem()
	kc := keychain.NewWithPath("clocked.keychain")
	if kc.Status() == keychain.ErrorNoSuchKeychain {
		kc, err = keychain.NewKeychainWithPrompt("clocked.keychain")
		if err != nil {
			return "", errors.Wrap(err, "failed to create new keychain")
		}
	}
	item.UseKeychain(kc)
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(url)
	item.SetAccount(username)
	item.SetMatchLimit(keychain.MatchLimitOne)
	item.SetReturnData(true)
	res, err := keychain.QueryItem(item)
	if err != nil {
		return "", errors.Wrap(err, "failed to query keychain")
	}
	if len(res) == 0 {
		pwd, err := speakeasy.Ask("JIRA password:")
		if err != nil {
			return "", errors.Wrap(err, "failed to read password from prompt")
		}
		addItem := keychain.NewItem()
		addItem.UseKeychain(kc)
		addItem.SetSecClass(keychain.SecClassGenericPassword)
		addItem.SetService(url)
		addItem.SetAccount(username)
		addItem.SetAccessible(keychain.AccessibleWhenUnlocked)
		addItem.SetData([]byte(pwd))
		if err := keychain.AddItem(addItem); err != nil {
			return "", errors.Wrap(err, "failed to add password to keychain")
		}
		return pwd, nil
	}
	return string(res[0].Data), nil
}
