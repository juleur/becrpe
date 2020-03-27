package utils

import (
	"errors"
)

// IPsChecker func
func IPsChecker(currentUserIP string, lastCachedUserIP string) error {
	if lastCachedUserIP == "" || currentUserIP == lastCachedUserIP {
		return nil
	}
	return errors.New("L'accès aux cours vidéo est autorisé sur seulement un appareil à la fois")
}
