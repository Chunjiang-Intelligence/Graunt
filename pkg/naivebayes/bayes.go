package naivebayes

import "sync"

type UserProfile struct {
	TruePositive  int // 实际对，评价对
	FalseNegative int // 实际对，评价错
	TrueNegative  int // 实际错，评价错
	FalsePositive int // 实际错，评价对
}

type BayesClassifier struct {
	mu       sync.RWMutex
	Profiles map[string]*UserProfile
}

func NewBayesClassifier() *BayesClassifier {
	return &BayesClassifier{
		Profiles: make(map[string]*UserProfile),
	}
}

func (bc *BayesClassifier) UpdateProfile(userID string, userEval, actualCorrect bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, exists := bc.Profiles[userID]; !exists {
		bc.Profiles[userID] = &UserProfile{}
	}

	p := bc.Profiles[userID]
	if actualCorrect && userEval {
		p.TruePositive++
	} else if actualCorrect && !userEval {
		p.FalseNegative++
	} else if !actualCorrect && !userEval {
		p.TrueNegative++
	} else if !actualCorrect && userEval {
		p.FalsePositive++
	}
}

func (bc *BayesClassifier) InferUnknownQuality(userID string, userEval bool) float64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	p, exists := bc.Profiles[userID]
	if !exists {
		return 0.5
	}

	tp := float64(p.TruePositive) + 1.0
	fn := float64(p.FalseNegative) + 1.0
	tn := float64(p.TrueNegative) + 1.0
	fp := float64(p.FalsePositive) + 1.0

	priorTrue := 0.5
	priorFalse := 0.5

	if userEval {
		// P(Eval=T | Real=T) = TP / (TP + FN)
		probEvalTrueGivenRealTrue := tp / (tp + fn)
		// P(Eval=T | Real=F) = FP / (FP + TN)
		probEvalTrueGivenRealFalse := fp / (fp + tn)

		numerator := probEvalTrueGivenRealTrue * priorTrue
		denominator := numerator + (probEvalTrueGivenRealFalse * priorFalse)
		return numerator / denominator
	} else {
		probEvalFalseGivenRealTrue := fn / (tp + fn)
		probEvalFalseGivenRealFalse := tn / (fp + tn)

		numerator := probEvalFalseGivenRealTrue * priorTrue
		denominator := numerator + (probEvalFalseGivenRealFalse * priorFalse)
		return numerator / denominator
	}
}