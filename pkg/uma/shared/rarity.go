package shared

import "fmt"

// GetRarityText converts numeric rarity to text representation
func GetRarityText(rarity int) string {
	switch rarity {
	case 3:
		return "SSR"
	case 2:
		return "SR"
	case 1:
		return "R"
	default:
		return fmt.Sprintf("Unknown(%d)", rarity)
	}
}
