package api

import (
	"net/http"
)

// testCore is a configurable Core implementation for tests.
// Set the response/code/error fields before invoking handlers/router.
type testCore struct {
	// capture
	startCalled   bool
	getCalled     bool
	stopCalled    bool
	resultsCalled bool

	// scripted returns
	tryStartResp StartJobResponse
	tryStartCode int
	tryStartErr  error

	getJobResp JobStatus
	getJobCode int
	getJobErr  error

	stopResp StopJobResponse
	stopCode int
	stopErr  error

	resultsResp JobResults
	resultsCode int
	resultsErr  error
}

func (t *testCore) TryStartJob(req StartJobRequest) (StartJobResponse, int, error) {
	t.startCalled = true
	code := t.tryStartCode
	if code == 0 {
		code = http.StatusCreated
	}
	return t.tryStartResp, code, t.tryStartErr
}

func (t *testCore) GetJob(id string) (JobStatus, int, error) {
	t.getCalled = true
	code := t.getJobCode
	if code == 0 {
		code = http.StatusOK
	}
	return t.getJobResp, code, t.getJobErr
}

func (t *testCore) StopJob(id string) (StopJobResponse, int, error) {
	t.stopCalled = true
	code := t.stopCode
	if code == 0 {
		code = http.StatusOK
	}
	return t.stopResp, code, t.stopErr
}

func (t *testCore) GetResults(id string) (JobResults, int, error) {
	t.resultsCalled = true
	code := t.resultsCode
	if code == 0 {
		code = http.StatusOK
	}
	return t.resultsResp, code, t.resultsErr
}
