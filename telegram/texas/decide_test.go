package texas

import (
	"fmt"
	"testing"
)

func TestDecideShowdownType(t *testing.T) {
	tests := []struct {
		name  string
		cards []string
		want  Hand
	}{
		// five cards Hand
		{
			name:  "Royal Flush", // 皇家同花顺不一定要黑桃, 任何颜色都可以
			cards: []string{"Spade 14", "Spade 13", "Spade 12", "Spade 11", "Spade 10"},
			want:  RoyalFlush,
		},
		{
			name:  "Straight FLush",
			cards: []string{"Spade 13", "Spade 12", "Spade 11", "Spade 10", "Spade 9"},
			want:  StraightFlush,
		},
		{
			name:  "Four of a Kind",
			cards: []string{"Spade 11", "Diamond 11", "Heart 11", "Club 11", "Spade 9"},
			want:  FourOfKind,
		},
		{
			name:  "Full House",
			cards: []string{"Spade 11", "Diamond 11", "Heart 11", "Club 9", "Spade 9"},
			want:  FullHouse,
		},
		{
			name:  "flush",
			cards: []string{"Spade 11", "Spade 7", "Spade 6", "Spade 5", "Spade 4"},
			want:  Flush,
		},
		{
			name:  "Straight",
			cards: []string{"Heart 8", "Spade 7", "Spade 6", "Spade 5", "Spade 4"},
			want:  Straight,
		},
		{
			name:  "Straight",
			cards: []string{"Heart 14", "Spade 5", "Spade 4", "Spade 3", "Spade 2"}, // 最小顺子
			want:  Straight,
		},
		{
			name:  "Three of a Kind",
			cards: []string{"Spade 11", "Diamond 11", "Heart 11", "Club 9", "Spade 8"},
			want:  ThreeOfKind,
		},
		{
			name:  "Two pair",
			cards: []string{"Spade 11", "Diamond 11", "Heart 8", "Club 8", "Spade 3"},
			want:  TwoPair,
		},
		{
			name:  "Pair",
			cards: []string{"Spade 11", "Diamond 11", "Heart 7", "Club 5", "Spade 3"},
			want:  Pair,
		},
		{
			name:  "High cards",
			cards: []string{"Spade 11", "Diamond 10", "Heart 7", "Club 5", "Spade 3"},
			want:  HighCard,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards, _ := CardsFromStrings(tt.cards)
			got := DecideShowdownType(cards)
			if got.Hand() != tt.want {
				t.Errorf("DecideShowdownType() = %v, want %v", got.Hand(), tt.want)
			}
		})
	}
}

func TestDecideShowDownTypeUnsorted(t *testing.T) {
	tests := []struct {
		name  string
		cards []string
		want  Hand
	}{
		// Seven cards table
		{
			name:  "Royal Flush",
			cards: []string{"Heart 13", "Heart 14", "Club 5", "Spade 6", "Heart 12", "Heart 11", "Heart 10"},
			want:  RoyalFlush,
		},
		{
			// "4h 3h 5h 6h 7h 8h Jc"
			name:  "Straight Flush",
			cards: []string{"Heart 4", "Heart 3", "Heart 5", "Heart 6", "Heart 7", "Heart 8", "Club 11"},
			want:  StraightFlush,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cards, _ := CardsFromStrings(tt.cards)
			got := DecideShowDownTypeUnsorted(cards)
			if got.Hand() != tt.want {
				t.Errorf("DecideShowDownTypeUnsorted() = %v, want %v", got.Hand(), tt.want)
			}
		})
	}
}

const C5From50 = 2118760

func TestHistogramHandTypes(t *testing.T) {
	tests := []struct {
		holeCards []string
		want      Hand
	}{
		// Seven cards table
		{
			holeCards: []string{"Heart 13", "Heart 14"},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.holeCards), func(t *testing.T) {
			holeCards, _ := CardsFromStrings(tt.holeCards)
			got := HistogramHandTypes(holeCards)
			sum := 0
			for i := range got {
				sum += got[i].Count
			}
			if sum != C5From50 {
				t.Errorf("expected %d simulations, got %d", C5From50, sum)
			}
		})
	}
}

func BenchmarkHistogramHandTypes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		holeCards, _ := CardsFromStrings([]string{"Heart 13", "Heart 14"})
		got := HistogramHandTypes(holeCards)
		sum := 0
		for i := range got {
			sum += got[i].Count
		}
	}
}
