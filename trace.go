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

// PrepTime returns the time taken to prepare and send the request to the upstream server
func (t *Trace) PrepTime() time.Duration { return t.timeToRequestToUpstream }

// WriteTime returns the time taken to write the response back to the client
func (t *Trace) WriteTime() time.Duration { return t.timeToResponseFromProxy }

// UpstreamLatency returns the time taken by the upstream server to process the request and send a response
func (t *Trace) UpstreamLatency() time.Duration { return t.timeToResponseFromUpstream }

// TotalDuration returns the total time taken from receiving the request to sending the response back to the client
func (t *Trace) TotalDuration() time.Duration { return t.timeToCompleteRequest }

// ProxyOverhead returns the time spent in the proxy itself (preparation + writing response)
func (t *Trace) ProxyOverhead() time.Duration { return t.proxyProcessingTime }

// IngestEndpoint returns the original request path that was ingested by the proxy
func (t *Trace) IngestEndpoint() string { return t.ingestEndpoint }

// OutgoingEndpoint returns the full URL that the proxy sent the request to upstream
func (t *Trace) OutgoingEndpoint() string { return t.outgoingEndpoint }

// WorkerPID returns the PID of the worker process that handled the request (prefork mode)
func (t *Trace) WorkerPID() int { return t.workerPID }

// WorkerID returns the ID of the worker process that handled the request (prefork mode)
func (t *Trace) WorkerID() int { return t.workerID }
