package tests

import (
	"strings"
	"testing"
	"time"

	apiv1 "github.com/MeowMeowOrganization/meow-contracts/gen/go/meow/exchange/api/v1alpha1"
	marketdatav1 "github.com/MeowMeowOrganization/meow-contracts/gen/go/meow/exchange/marketdata/v1alpha1"
	matchingv1 "github.com/MeowMeowOrganization/meow-contracts/gen/go/meow/exchange/matching/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Compatibility guard: fields used before CEX 23 may NEVER be renumbered,
// renamed or change wire kinds in the existing v1alpha1 TradeTick.
func TestTradeTickLegacyFieldNumbers(t *testing.T) {
	fields := (&marketdatav1.TradeTick{}).ProtoReflect().Descriptor().Fields()
	for _, tc := range []struct {
		name   protoreflect.Name
		number protoreflect.FieldNumber
		kind   protoreflect.Kind
	}{
		{"trade_id", 1, protoreflect.StringKind},
		{"market_id", 2, protoreflect.StringKind},
		{"price_units", 3, protoreflect.Uint64Kind},
		{"base_quantity_atoms", 4, protoreflect.Uint64Kind},
		{"quote_quantity_atoms", 5, protoreflect.Uint64Kind},
		{"aggressor_side", 6, protoreflect.EnumKind},
		{"match_sequence", 7, protoreflect.Uint64Kind},
	} {
		f := fields.ByName(tc.name)
		if f == nil || f.Number() != tc.number || f.Kind() != tc.kind {
			t.Fatalf("legacy TradeTick.%s changed: got %v; want number=%d kind=%s", tc.name, f, tc.number, tc.kind)
		}
	}
}

func TestTimestampedEnvelopePreservesZeroEventPlacement(t *testing.T) {
	in := &matchingv1.CommittedMatchingResult{
		Result: &matchingv1.MatchingResult{
			Command: &matchingv1.MatchingCommand{
				CommandId: "c-123", MarketId: 123,
				Command: &matchingv1.MatchingCommand_PlaceLimitOrder{
					PlaceLimitOrder: &matchingv1.PlaceLimitOrder{OrderId: 9001, PriceTicks: 65000, QuantityLots: 5},
				},
			},
			MarketSequence: 1,
		},
		RecordedAt:              timestamppb.New(time.Date(2026, time.November, 10, 12, 0, 0, 0, time.UTC)),
		MarketDefinitionVersion: 9,
	}
	encoded, err := proto.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out matchingv1.CommittedMatchingResult
	if err := proto.Unmarshal(encoded, &out); err != nil {
		t.Fatal(err)
	}
	if out.GetResult().GetMarketSequence() != 1 || len(out.GetResult().GetEvents()) != 0 ||
		out.GetResult().GetCommand().GetPlaceLimitOrder().GetOrderId() != 9001 ||
		out.GetMarketDefinitionVersion() != 9 || !out.GetRecordedAt().AsTime().Equal(in.GetRecordedAt().AsTime()) {
		t.Fatalf("zero-event placement or envelope metadata was lost: %+v", &out)
	}
}

func TestBookNoopDeltaAndLargeExactAggregation(t *testing.T) {
	delta := &marketdatav1.BookDelta{MarketId: "BTC-USDT", SourceMarketId: 123,
		PreviousMarketSequence: 1, MarketSequence: 2, MarketDefinitionVersion: 9, Depth: 20}
	bytes, err := proto.Marshal(delta)
	if err != nil {
		t.Fatal(err)
	}
	var restored marketdatav1.BookDelta
	if err := proto.Unmarshal(bytes, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.GetMarketSequence() != 2 || restored.GetPreviousMarketSequence() != 1 ||
		len(restored.GetBidUpdates()) != 0 || len(restored.GetAskUpdates()) != 0 {
		t.Fatalf("no-op book delta must still advance sequence: %+v", &restored)
	}

	huge := "184467440737095516160000"
	level := &marketdatav1.BookLevel{PriceUnits: 65000, BaseQuantityAtoms: huge}
	wire, err := proto.Marshal(level)
	if err != nil {
		t.Fatal(err)
	}
	var levelAfter marketdatav1.BookLevel
	if err := proto.Unmarshal(wire, &levelAfter); err != nil {
		t.Fatal(err)
	}
	if levelAfter.GetBaseQuantityAtoms() != huge {
		t.Fatalf("aggregate precision lost: %s", levelAfter.GetBaseQuantityAtoms())
	}
}

func TestTradeOrdinalAndTimestampRoundTrip(t *testing.T) {
	ticks := []*marketdatav1.TradeTick{
		{TradeId: "123:99:0", MarketId: "BTC-USDT", MatchSequence: 99, EventOrdinal: 0,
			SourceMarketId: 123, MarketDefinitionVersion: 9,
			RecordedAt: timestamppb.New(time.Unix(1000, 0))},
		{TradeId: "123:99:1", MarketId: "BTC-USDT", MatchSequence: 99, EventOrdinal: 1,
			SourceMarketId: 123, MarketDefinitionVersion: 9,
			RecordedAt: timestamppb.New(time.Unix(1000, 0))},
	}
	for i, tick := range ticks {
		wire, err := proto.Marshal(tick)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip marketdatav1.TradeTick
		if err := proto.Unmarshal(wire, &roundTrip); err != nil {
			t.Fatal(err)
		}
		if roundTrip.GetEventOrdinal() != uint64(i) || roundTrip.GetSourceMarketId() != 123 ||
			roundTrip.GetRecordedAt().GetSeconds() != 1000 {
			t.Fatalf("trade %d lost event identity/timestamp", i)
		}
	}
	if ticks[0].GetTradeId() == ticks[1].GetTradeId() {
		t.Fatal("distinct executions must have distinct trade IDs")
	}
}

func TestOptionalTickerPriceAndMarketDataStreamOneof(t *testing.T) {
	tick := &marketdatav1.Ticker{MarketId: "BTC-USDT", LastPriceUnits: proto.Uint64(0)}
	if tick.LastPriceUnits == nil || tick.BestBidPriceUnits != nil {
		t.Fatal("zero price presence must differ from absent price")
	}
	update := &marketdatav1.MarketDataUpdate{Update: &marketdatav1.MarketDataUpdate_BookDelta{
		BookDelta: &marketdatav1.BookDelta{MarketSequence: 5, PreviousMarketSequence: 4},
	}}
	data, err := proto.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	var decoded marketdatav1.MarketDataUpdate
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetBookDelta().GetMarketSequence() != 5 || decoded.GetTrade() != nil {
		t.Fatal("stream oneof changed variant")
	}
}

func TestPublicMarketDataRequestsNeverAcceptAccountID(t *testing.T) {
	// Reflection test ensures no accidental account identity is added to a
	// public market-data request contract during later changes.
	messages := []proto.Message{
		&apiv1.GetOrderBookRequest{}, &apiv1.ListRecentTradesRequest{},
		&apiv1.GetTickerRequest{}, &apiv1.ListCandlesRequest{}, &apiv1.GetStats24HRequest{},
	}
	for _, m := range messages {
		fields := m.ProtoReflect().Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			name := string(fields.Get(i).Name())
			if strings.Contains(name, "account") || strings.Contains(name, "user") {
				t.Fatalf("public request %s exposes private identifier %s", m.ProtoReflect().Descriptor().FullName(), name)
			}
		}
	}
}
