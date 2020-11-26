package openrtb_ext

// ExtImpNanoInteractive defines the contract for bidrequest.imp[i].ext.nanointeractive
type ExtImpNanoInteractive struct {
	// Pid is optional parameter when NetworkId (Nid) is provided
	// Identifies pixed id
	Pid string `json:"pid,omitempty"`
	// Nid is optional parameter when PublisherId (Pid) is provided
	Nid      string   `json:"nid,omitempty"`
	Nq       []string `json:"nq,omitempty"`
	Category string   `json:"category,omitempty"`
	SubId    string   `json:"subId,omitempty"`
	Ref      string   `json:"ref,omitempty"`
}
