package nanointeractive

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

var ErrNoImpressionsInBid = errors.New("no impressions in the bid request")

type NanoInteractiveAdapter struct {
	endpoint string
}

func (a *NanoInteractiveAdapter) Name() string {
	return "Nano"
}

func (a *NanoInteractiveAdapter) SkipNoCookies() bool {
	return false
}

func (a *NanoInteractiveAdapter) MakeRequests(bidRequest *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) (adapterRequests []*adapters.RequestData, errs []error) {
	validImps := make([]openrtb.Imp, 0, len(bidRequest.Imp))

	referer := ""
	for _, impl := range bidRequest.Imp {

		ref, err := checkImp(&impl)

		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if referer == "" && ref != "" {
			referer = ref
		}
		validImps = append(validImps, impl)
	}

	if len(validImps) == 0 {
		errs = append(errs, ErrNoImpressionsInBid)
		return nil, errs
	}

	// set referer origin
	if referer != "" {
		if bidRequest.Site == nil {
			bidRequest.Site = &openrtb.Site{}
		}
		bidRequest.Site.Ref = referer
	}

	bidRequest.Imp = validImps

	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	adapterRequests = append(adapterRequests, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: createHeaders(bidRequest),
	})

	return adapterRequests, errs
}

func (a *NanoInteractiveAdapter) MakeBids(
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if response.StatusCode == http.StatusBadRequest {
		return nil, []error{adapters.BadInput("Invalid request.")}
	} else if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected HTTP status %d.", response.StatusCode),
		}}
	}

	var openRtbBidResponse openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &openRtbBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server body response"),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(openRtbBidResponse.SeatBid[0].Bid))
	bidResponse.Currency = openRtbBidResponse.Cur

	sb := openRtbBidResponse.SeatBid[0]
	for _, bid := range sb.Bid {
		if bid.Price > 0 {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}
	return bidResponse, nil
}

func checkImp(imp *openrtb.Imp) (string, error) {
	// We support only banner impression
	if imp.Banner == nil {
		return "", fmt.Errorf("invalid MediaType. NanoInteractive only supports Banner type. ImpID=%s", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", fmt.Errorf("ext not provided; ImpID=%s", imp.ID)
	}

	var nanoExt openrtb_ext.ExtImpNanoInteractive
	if err := json.Unmarshal(bidderExt.Bidder, &nanoExt); err != nil {
		return "", fmt.Errorf("ext.bidder not provided; ImpID=%s", imp.ID)
	}

	if nanoExt.Pid == "" && nanoExt.Nid == "" {
		return "", fmt.Errorf("pid and nid are empty, one of them must be provided; ImpID=%s", imp.ID)
	}

	if nanoExt.Ref != "" {
		return nanoExt.Ref, nil
	}

	return "", nil
}

func createHeaders(bidRequest *openrtb.BidRequest) http.Header {
	values := [3]string{
		"application/json;charset=utf-8", // Content-Type
		"application/json",               // Accept
		"2.5",                            // OpenRTB Version
	}

	headers := http.Header{
		"Content-Type":      values[0:1],
		"Accept":            values[1:2],
		"X-Openrtb-Version": values[2:3],
	}

	if bidRequest.Device != nil {
		headers["User-Agent"] = []string{bidRequest.Device.UA}
		headers["X-Forwarded-IP"] = []string{bidRequest.Device.IP}
	}

	if bidRequest.Site != nil {
		headers["Referer"] = []string{bidRequest.Site.Page}
	}

	// set user's cookie
	if bidRequest.User != nil && bidRequest.User.BuyerUID != "" {
		headers["Cookie"] = []string{"Nano=" + bidRequest.User.BuyerUID}
	}

	return headers
}

func NewNanoIneractiveBidder(endpoint string) *NanoInteractiveAdapter {
	return &NanoInteractiveAdapter{
		endpoint: endpoint,
	}
}

func NewNanoInteractiveAdapter(uri string) *NanoInteractiveAdapter {
	return &NanoInteractiveAdapter{
		endpoint: uri,
	}
}
