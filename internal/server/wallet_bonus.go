package server

import (
	"regexp"
	"strconv"
)

var walletPercent = regexp.MustCompile(`^(\d{1,4})(?:\.(\d{1,2}))?$`)

// Percentage accepts up to two decimal places and rounds half-up to a cent.
func walletRechargeBonus(principal int64, mode, value string) (int64, bool) {
	switch mode {
	case "none":
		return 0, value == ""
	case "fixed":
		return parseWalletAmount(value)
	case "percent":
		match := walletPercent.FindStringSubmatch(value)
		if match == nil {
			return 0, false
		}
		whole, _ := strconv.ParseInt(match[1], 10, 64)
		fraction := match[2]
		if len(fraction) == 1 {
			fraction += "0"
		}
		decimal, _ := strconv.ParseInt(fraction, 10, 64)
		basisPoints := whole*100 + decimal
		if basisPoints < 1 || basisPoints > 100000 {
			return 0, false
		}
		bonus := (principal*basisPoints + 5000) / 10000
		return bonus, bonus > 0
	default:
		return 0, false
	}
}
