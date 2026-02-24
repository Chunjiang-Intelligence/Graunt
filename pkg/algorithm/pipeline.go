package service

import "graunt/internal/model"

type PipelineResult struct {
	Passed bool
	Reason string
}

func ExecuteFilterPipeline(req model.PipelineFilterRequest) PipelineResult {
	for _, algoName := range req.Algorithms {
		algo, err := GetAlgorithm(algoName)
		if err != nil {
			return PipelineResult{Passed: false, Reason: err.Error()}
		}

		keep, reason := algo.Evaluate(req.Text, req.Params)
		if !keep {
			return PipelineResult{
				Passed: false,
				Reason: "Failed at " + algoName + ": " + reason,
			}
		}
	}

	return PipelineResult{Passed: true, Reason: "Passed all filters"}
}