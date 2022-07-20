package texas

import (
	"fmt"
	"gonum.org/v1/gonum/stat/combin"
	"sort"
	"strconv"
	"sync"
)

func CardsFromStrings(s []string) ([]Card, error) {
	var cs []Card
	for _, s := range s {
		c, err := NewCardFromString(s)
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
	}
	return cs, nil
}

func sortCards(cards []Card, asc bool) {
	sort.Slice(cards, func(i, j int) bool {
		if asc {
			return cards[i].Rank < cards[j].Rank
		} else {
			return cards[i].Rank > cards[j].Rank
		}
	})
}

func NewCardFromString(s string) (Card, error) {
	var (
		suiteName string
		rankStr   string
		c         Card
		rank      Rank
	)
	_, err := fmt.Sscanf(s, "%s %s", &suiteName, &rankStr)
	if err != nil {
		return c, fmt.Errorf("cannot scan input")
	}
	// short name is supported
	switch suiteName {
	case "Diamond", "D":
		c.Suite = Diamond
	case "Club", "C":
		c.Suite = Club
	case "Heart", "H":
		c.Suite = Heart
	case "Spade", "S":
		c.Suite = Spade
	default:
		return c, fmt.Errorf("invalid suite: %v", suiteName)
	}

	switch rankStr {
	case "J":
		rank = 11
	case "Q":
		rank = 12
	case "K":
		rank = 13
	case "A":
		rank = Ace
	default:
		// try strconv
		rankI, err := strconv.Atoi(rankStr)
		if err != nil {
			return c, fmt.Errorf("invalid rank: %v", err)
		}
		rank = Rank(rankI)
	}
	c.Rank = rank
	if c.Rank < 2 || c.Rank > 14 {
		return c, fmt.Errorf("invalid rank: %v", c.Rank)
	}
	return c, nil
}

func DecideShowDownTypeUnsorted(cards []Card) HandCards {
	sortCards(cards, false)
	return DecideShowdownType(cards)
}

// DecideShowdownType Given seven cards, sorted desc, returns the best hand of five cards
// one can hold
func DecideShowdownType(cards []Card) HandCards {
	bestSuite, bestSuiteCount := findBestSuite(cards)
	if bestSuiteCount >= 5 {
		// 最低同花
		// full house, four of a kind is not possible (at least five cards are of different ranks, )

		// optimal play could be flush / straight / royal flush
		flushCards := findCardsInSuite(cards, bestSuite)
		if sf := isStraight(flushCards); sf != nil {
			// 最低同花顺
			if sf[0].Rank == Ace {
				return RoyalFlushCards{} // 皇家同花顺
			} else {
				return StraightFlushCards{
					&StraightCards{
						bestRank: sf[0].Rank,
					},
				} // 同花顺
			}
		} else {
			// 同花
			return FlushCards{
				NewHighCardCards(sf),
			}
		}
	}

	// Trick: find most frequent rank and second most frequent rank
	rankCountSorted := sortByRankCount(cards)
	best, secondBest := rankCountSorted[len(rankCountSorted)-1], rankCountSorted[len(rankCountSorted)-2]

	if best.count == 4 {
		// 保底four of a kind
		return FourOfKindCards{
			fourRank: best.rank,
			// find the best kicker, 挑选一张不是4张牌rank的最大的牌
			kicker: *matchCard(cards, func(card Card) bool {
				return card.Rank != best.rank
			}),
		}
	}

	// could be 3, 3 or 3, 2
	// the (3, 3) case is why we need to count cards also by rank to determine which one is triplet (the major rank)
	if best.count == 3 && secondBest.count >= 2 {
		return FullHouseCards{
			tripletRank: best.rank,
			twinRank:    secondBest.rank,
		}
	}

	// check if it is possible to have a straight
	if len(rankCountSorted) >= 5 {
		if straight := isStraight(cards); straight != nil {
			return StraightCards{
				bestRank: straight[0].Rank,
			}
		}
	}

	// check if there is a three of a kind
	if best.count == 3 {
		return ThreeOfKindCards{
			tripletRank: best.rank,
			kickers: matchNCards(cards, func(card Card) bool {
				return card.Rank != best.rank
			}, 2),
		}
	}

	if best.count == 2 {
		// check if two pair or pair
		strongPairRank := best.rank
		if secondBest.count == 2 {
			// two pair
			return TwoPairCards{
				strongRank: strongPairRank,
				weakRank:   secondBest.rank,
				// find the best kicker, 挑选不是大对子rank而且不是小对子rank的最大牌
				kicker: *matchCard(cards, func(card Card) bool {
					return card.Rank != strongPairRank && card.Rank != secondBest.rank
				}),
			}
		}

		return PairCards{
			pairRank: best.rank,
			// find the best kickers, 挑选不是对子rank的最大3张牌
			kickers: matchNCards(cards, func(card Card) bool {
				return card.Rank != best.rank
			}, 3),
		}

	}

	// 全部五张牌中挑最大的五张牌
	return NewHighCardCards(matchNCards(cards, func(card Card) bool {
		return true
	}, 5))
}

type HandCardsProbability struct {
	Hand
	Prob    float64
	AccProb float64 // cumulated probability of getting something better
	Count   int
}

// GetAllCards 生成54张一副扑克牌
func GetAllCards() []Card {
	var acc []Card
	for rank := 2; rank <= 14; rank++ {
		for _, suite := range []Suite{
			Diamond,
			Club,
			Heart,
			Spade,
		} {
			acc = append(acc, Card{
				Rank:  Rank(rank),
				Suite: suite,
			})
		}
	}
	return acc
}

func CardIn(cards []Card) func(Card) bool {
	return func(card Card) bool {
		for i := range cards {
			if cards[i] == card {
				return true
			}
		}
		return false
	}
}

func CardNotIn(cards []Card) func(Card) bool {
	return func(card Card) bool {
		return !CardIn(cards)(card)
	}
}

func GetRemainingCards(known []Card) []Card {
	return matchNCards(GetAllCards(), CardNotIn(known), -1)
}

// HistogramHandTypes 计算蒙特卡洛概率
// performance
func HistogramHandTypes(known []Card) []HandCardsProbability {
	var distributionLock sync.Mutex
	distribution := make(map[Hand]int)
	remaining := GetRemainingCards(known)
	// 选五张table cards
	leftToShow := 7 - len(known)
	combs := combin.Combinations(len(remaining), leftToShow)
	var wg sync.WaitGroup
	for _, combIndices := range combs {
		combIndices := combIndices
		wg.Add(1)
		go func() {
			defer wg.Done()
			// pick cards
			cards := make([]Card, len(known))
			copy(cards, known)
			for _, idx := range combIndices {
				cards = append(cards, remaining[idx])
			}
			handType := DecideShowDownTypeUnsorted(cards)
			distributionLock.Lock()
			distribution[handType.Hand()]++
			distributionLock.Unlock()
		}()
	}
	wg.Wait()

	probSlice := make([]HandCardsProbability, 0)
	accProb := 0.0
	for _, h := range []Hand{
		RoyalFlush,
		StraightFlush,
		FourOfKind,
		FullHouse,
		Flush,
		Straight,
		ThreeOfKind,
		TwoPair,
		Pair,
		HighCard,
	} {
		p := float64(distribution[h]) / float64(len(combs))
		accProb += p
		probSlice = append(probSlice, HandCardsProbability{
			Hand:    h,
			Prob:    p,
			Count:   distribution[h],
			AccProb: accProb,
		})
	}
	return probSlice
}

type rankCount struct {
	rank  Rank
	count int
}

func matchCard(cards []Card, f func(Card) bool) *Card {
	for i := range cards {
		if f(cards[i]) {
			c := cards[i]
			return &c
		}
	}
	return nil
}

// provide negative count to pick all cards that are matched
func matchNCards(cards []Card, f func(Card) bool, count int) []Card {
	var acc []Card
	var c int
	for i := range cards {
		if f(cards[i]) {
			c++
			acc = append(acc, cards[i])
			if c == count {
				return acc
			}
		}
	}
	if count >= 0 {
		panic("didn't pick enough cards")
	}
	return acc
}

func sortByRankCount(cards []Card) []rankCount {
	rcs := make([]rankCount, 0)
	m := make(map[Rank]int)
	for i := range cards {
		m[cards[i].Rank] += 1
	}

	for r, c := range m {
		rcs = append(rcs, rankCount{
			rank:  r,
			count: c,
		})
	}
	sort.Slice(rcs, func(i, j int) bool {
		if rcs[i].count != rcs[j].count {
			return rcs[i].count < rcs[j].count
		}
		return rcs[i].rank < rcs[j].rank
	})
	return rcs
}

// isStraight 判断是否是顺子
// 如果是返回五张牌顺子, 按rank大小排序desc
func isStraight(cards []Card) []Card {
	cursor := cards[0]
	acc := []Card{cursor}

	for i, card := range cards {
		if i > 0 {
			if cursor.Rank-1 == card.Rank {
				cursor = card
				acc = append(acc, card)
				if len(acc) == 5 {
					return acc
				}
			} else if cursor.Rank == card.Rank {
			} else if cursor.Rank-1 > card.Rank {
				// broken straight, reset acc
				cursor = card
				acc = []Card{cursor}
			}
		}
	}

	// handle Special A2345 straight (the smallest straight)
	if cursor.Rank == 2 && len(acc) == 4 {
		if c := matchCard(cards, func(card Card) bool {
			return card.Rank == Ace
		}); c != nil {
			acc = append(acc, *c) // Ace 是最小的一张牌
			return acc
		}
	}
	return nil
}

func findCardsInSuite(cards []Card, s Suite) []Card {
	sameSuite := make([]Card, 0)
	for i := range cards {
		if cards[i].Suite == s {
			sameSuite = append(sameSuite, cards[i])
		}
	}
	return sameSuite
}

func findBestSuite(cards []Card) (Suite, int) {
	m := make(map[Suite]int)
	for i := range cards {
		m[cards[i].Suite] += 1
	}

	var (
		bs      Suite
		bsCount int
	)

	for s, c := range m {
		if c > bsCount {
			bs = s
			bsCount = c
		}
	}
	return bs, bsCount

}
