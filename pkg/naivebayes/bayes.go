package naivebayes

import "sync"

type UserProfile struct {
	TruePositive  int
	FalseNegative int
	TrueNegative  int
	FalsePositive int
}

type BayesClassifier struct {
	mu       sync.RWMutex
	Profiles map[string]*UserProfile
}

func NewBayesClassifier() *BayesClassifier {
	return &BayesClassifier{Profiles: make(map[string]*UserProfile)}
}

func (bc *BayesClassifier) UpdateProfile(userID string, userEval, actualCorrect bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, exists := bc.Profiles[userID]; !exists {
		bc.Profiles[userID] = &UserProfile{}
	}

	p := bc.Profiles[userID]
	if actualCorrect && userEval { p.TruePositive++ }
	if actualCorrect && !userEval { p.FalseNegative++ }
	if !actualCorrect && !userEval { p.TrueNegative++ }
	if !actualCorrect && userEval { p.FalsePositive++ }
}

func (bc *BayesClassifier) InferUnknownQuality(userID string, userEval bool) float64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	p, exists := bc.Profiles[userID]
	if !exists { return 0.5 }

	tp := float64(p.TruePositive) + 1.0
	fn := float64(p.FalseNegative) + 1.0
	tn := float64(p.TrueNegative) + 1.0
	fp := float64(p.FalsePositive) + 1.0

	priorTrue := 0.5
	priorFalse := 0.5

	if userEval {
		probTrueGivenTrue := tp / (tp + fn)
		probTrueGivenFalse := fp / (fp + tn)
		num := probTrueGivenTrue * priorTrue
		den := num + (probTrueGivenFalse * priorFalse)
		return num / den
	} else {
		probFalseGivenTrue := fn / (tp + fn)
		probFalseGivenFalse := tn / (fp + tn)
		num := probFalseGivenTrue * priorTrue
		den := num + (probFalseGivenFalse * priorFalse)
		return num / den
	}
}