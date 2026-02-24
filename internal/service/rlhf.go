package service

import (
	"graunt/pkg/naivebayes"
)

type RLHFService struct {
	Classifier *naivebayes.BayesClassifier
}

func NewRLHFService() *RLHFService {
	return &RLHFService{
		Classifier: naivebayes.NewBayesClassifier(),
	}
}

func (s *RLHFService) SubmitKnownEvaluation(userID string, userEval, actualCorrect bool) {
	s.Classifier.UpdateProfile(userID, userEval, actualCorrect)
}

func (s *RLHFService) InferUnknownEvaluation(userID string, userEval bool) float64 {
	return s.Classifier.InferUnknownQuality(userID, userEval)
}