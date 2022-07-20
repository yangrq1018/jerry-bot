package texas

import "strconv"

type Suite int

func (s Suite) String() string {
	switch s {
	case Spade:
		return "spade"
	case Heart:
		return "heart"
	case Club:
		return "club"
	case Diamond:
		return "diamond"
	default:
		return ""
	}
}

const (
	Diamond Suite = 1 + iota
	Club
	Heart
	Spade
)

type Rank int

func (r Rank) String() string {
	if r <= 10 {
		return strconv.Itoa(int(r))
	} else {
		switch r {
		case Jack:
			return "Jack"
		case Queen:
			return "Queen"
		case King:
			return "King"
		case Ace:
			return "Ace"
		default:
			return ""
		}
	}
}

const (
	Ace   Rank = 14
	King  Rank = 13
	Queen Rank = 12
	Jack  Rank = 11
)

type Card struct {
	Suite
	Rank
}

func (c Card) String() string {
	return c.Suite.String() + " " + c.Rank.String()
}

type Hand int

// hand order: the higher the better
const (
	HighCard Hand = 1 + iota
	Pair
	TwoPair
	ThreeOfKind
	Straight
	Flush
	FullHouse
	FourOfKind
	StraightFlush
	RoyalFlush
)

func (h Hand) String() string {
	switch h {
	case HighCard:
		return "High Card"
	case Pair:
		return "Pair"
	case TwoPair:
		return "Two Pair"
	case ThreeOfKind:
		return "Three of a Kind"
	case Straight:
		return "Straight"
	case Flush:
		return "Flush"
	case FullHouse:
		return "Full House"
	case FourOfKind:
		return "Four of a Kind"
	case StraightFlush:
		return "Straight Flush"
	case RoyalFlush:
		return "Royal Flush"
	default:
		return ""
	}
}

type HandCards interface {
	Hand() Hand
}

type PairCards struct {
	pairRank Rank
	kickers  []Card
}

func (p PairCards) Hand() Hand {
	return Pair
}

type FourOfKindCards struct {
	fourRank Rank
	kicker   Card
}

func (f FourOfKindCards) Hand() Hand {
	return FourOfKind
}

type FullHouseCards struct {
	tripletRank Rank
	twinRank    Rank
}

func (f FullHouseCards) Hand() Hand {
	return FullHouse
}

type ThreeOfKindCards struct {
	tripletRank Rank
	kickers     []Card // 2
}

func (t ThreeOfKindCards) Hand() Hand {
	return ThreeOfKind
}

type TwoPairCards struct {
	strongRank Rank
	weakRank   Rank
	kicker     Card
}

func (t TwoPairCards) Hand() Hand {
	return TwoPair
}

type RoyalFlushCards struct{}

func (r RoyalFlushCards) Hand() Hand {
	return RoyalFlush
}

// StraightFlushCards
// 同花顺按顺子比较大小(只比较rank, 花色不计)
type StraightFlushCards struct {
	*StraightCards
}

func (s StraightFlushCards) Hand() Hand {
	return StraightFlush
}

// StraightCards
// 顺子之间比较最高rank
type StraightCards struct {
	bestRank Rank
}

func (s StraightCards) Hand() Hand {
	return Straight
}

// FlushCards
// 同花按照高牌比较大小
type FlushCards struct {
	*HighCardCards
}

func (f FlushCards) Hand() Hand {
	return Flush
}

type HighCardCards struct {
	ranks []Rank // sorted desc
}

func NewHighCardCards(cards []Card) *HighCardCards {
	hcc := new(HighCardCards)
	for i := range cards {
		hcc.ranks = append(hcc.ranks, cards[i].Rank)
	}
	return hcc
}

func (h HighCardCards) Hand() Hand {
	return HighCard
}
