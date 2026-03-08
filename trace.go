package hyproxia

import "time"

func (p *Proxy) SetTraceHandler(handler func(*Trace)) {
	if !p.config.EnableTracing {
		return
	}
	p.traceHandler = handler
}

func (p *Proxy) OnTrace(fn func(*Trace)) {
	if !p.config.EnableTracing {
		return
	}
	p.traceHandler = fn
}

// buildTrace calculates trace timings and populates the Trace struct
func buildTrace(ts traceTimestamps, ingest, outgoing string, t *Trace, workerID, workerPID int) {
	total := ts.t3.Sub(ts.t0)
	upstream := ts.t2.Sub(ts.t1)
	prepTime := ts.t1.Sub(ts.t0)
	writeTime := ts.t3.Sub(ts.t2)

	t.ingestEndpoint = ingest
	t.outgoingEndpoint = outgoing
	t.timeToRequestToUpstream = prepTime
	t.timeToResponseFromUpstream = upstream
	t.timeToResponseFromProxy = writeTime
	t.timeToCompleteRequest = total
	t.proxyProcessingTime = prepTime + writeTime
	t.workerID = workerID
	t.workerPID = workerPID
}

// Call functions to access trace data
func (t *Trace) PrepTime() time.Duration        { return t.timeToRequestToUpstream }
func (t *Trace) WriteTime() time.Duration       { return t.timeToResponseFromProxy }
func (t *Trace) UpstreamLatency() time.Duration { return t.timeToResponseFromUpstream }
func (t *Trace) TotalDuration() time.Duration   { return t.timeToCompleteRequest }
func (t *Trace) ProxyOverhead() time.Duration   { return t.proxyProcessingTime }
func (t *Trace) IngestEndpoint() string         { return t.ingestEndpoint }
func (t *Trace) OutgoingEndpoint() string       { return t.outgoingEndpoint }
func (t *Trace) WorkerPID() int                 { return t.workerPID }
func (t *Trace) WorkerID() int                  { return t.workerID }
