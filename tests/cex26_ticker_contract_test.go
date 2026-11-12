package tests

import (
	"testing"

	api "github.com/MeowMeowOrganization/meow-contracts/gen/go/meow/exchange/api/v1alpha1"
	md "github.com/MeowMeowOrganization/meow-contracts/gen/go/meow/exchange/marketdata/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestStreamTickerIsPublicServerStreamingWithLatestStateVariants(t *testing.T) {
	file := (&api.StreamTickerRequest{}).ProtoReflect().Descriptor().ParentFile()
	service := file.Services().ByName("MarketDataService")
	if service == nil {
		t.Fatal("public MarketDataService missing")
	}
	method := service.Methods().ByName("StreamTicker")
	if method == nil || !method.IsStreamingServer() || method.IsStreamingClient() ||
		method.Input().FullName() != "meow.exchange.api.v1alpha1.StreamTickerRequest" ||
		method.Output().FullName() != "meow.exchange.api.v1alpha1.StreamTickerResponse" {
		t.Fatalf("unexpected StreamTicker descriptor: %v", method)
	}
	for _, name := range []protoreflect.Name{"StreamOrderBook", "StreamTrades"} {
		old := service.Methods().ByName(name)
		if old == nil || !old.IsStreamingServer() {
			t.Fatalf("regressed stream method %s", name)
		}
	}
	req := (&api.StreamTickerRequest{}).ProtoReflect().Descriptor()
	if req.Fields().Len() != 1 || req.Fields().ByName("market_id").Number() != 1 {
		t.Fatalf("ticker request must contain only public market_id: %v", req.Fields())
	}
	res := (&api.StreamTickerResponse{}).ProtoReflect().Descriptor()
	if res.Oneofs().Len() != 1 || res.Oneofs().Get(0).Fields().Len() != 3 {
		t.Fatal("ticker response must have three disjoint event variants")
	}
	for _, name := range []protoreflect.Name{"snapshot", "update", "heartbeat"} {
		field := res.Fields().ByName(name)
		if field == nil || field.ContainingOneof() == nil {
			t.Fatalf("missing oneof event %s", name)
		}
	}
	if f := res.Fields().ByName("market_definition_version"); f == nil || f.Kind() != protoreflect.Uint64Kind {
		t.Fatal("market definition version missing from ticker envelope")
	}
	for _, m := range []proto.Message{&api.StreamTickerRequest{}, &api.StreamTickerResponse{}} {
		fields := m.ProtoReflect().Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			if fields.Get(i).Name() == "account_id" || fields.Get(i).Name() == "order_id" || fields.Get(i).Name() == "user_id" {
				t.Fatalf("private field in public ticker contract: %s", fields.Get(i).Name())
			}
		}
	}
}

func TestStreamTickerOptionalPricesRemainAbsent(t *testing.T) {
	in := &api.StreamTickerResponse{MarketDefinitionVersion: 3,
		Event: &api.StreamTickerResponse_Snapshot{Snapshot: &md.Ticker{MarketId: "BTC-USDT", MarketSequence: 1}}}
	wire, err := proto.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out api.StreamTickerResponse
	if err := proto.Unmarshal(wire, &out); err != nil {
		t.Fatal(err)
	}
	if out.GetSnapshot() == nil || out.GetSnapshot().LastPriceUnits != nil || out.GetSnapshot().BestAskPriceUnits != nil ||
		out.GetSnapshot().BestBidPriceUnits != nil || out.GetMarketDefinitionVersion() != 3 {
		t.Fatalf("absent prices must not become synthetic zeros: %v", &out)
	}
}
