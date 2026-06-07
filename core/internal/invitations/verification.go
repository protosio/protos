package invitations

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode"
)

const verificationCodeDigits = 8

func GenerateVerificationCode() (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(verificationCodeDigits), nil)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate invite verification code: %w", err)
	}
	return fmt.Sprintf("%0*d", verificationCodeDigits, value.Int64()), nil
}

func InviteVerificationHash(invite Invite, code string) string {
	return verificationHash(invite.InviteID, invite.OrganisationID, invite.PeerID, invite.PublicKey, invite.SwarmionAddrs, code)
}

func NearbyInviteVerificationHash(invite NearbyInvite, code string) string {
	return verificationHash(invite.InviteID, invite.OrganisationID, invite.PeerID, invite.PublicKey, invite.SwarmionAddrs, code)
}

func VerifyNearbyInviteCode(invite NearbyInvite, code string) error {
	if strings.TrimSpace(invite.VerificationHash) == "" {
		return fmt.Errorf("nearby invite does not include a verification hash")
	}
	got := NearbyInviteVerificationHash(invite, code)
	if subtle.ConstantTimeCompare([]byte(got), []byte(invite.VerificationHash)) != 1 {
		return fmt.Errorf("invite verification code is incorrect")
	}
	return nil
}

func NormalizeVerificationCode(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func verificationHash(inviteID string, organisationID string, peerID string, publicKey string, swarmionAddrs []string, code string) string {
	normalizedCode := NormalizeVerificationCode(code)
	fields := []string{
		"protos-local-invite-v1",
		strings.TrimSpace(inviteID),
		strings.TrimSpace(organisationID),
		strings.TrimSpace(peerID),
		strings.TrimSpace(publicKey),
	}
	fields = append(fields, normalizedStrings(swarmionAddrs)...)
	fields = append(fields, normalizedCode)
	payload := strings.Join(fields, "\n")
	sum := sha256.Sum256([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
