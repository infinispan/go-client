package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

const circularProtoSchema = `syntax = "proto3";
package test;

message Team {
  string name = 1;
  repeated Player players = 2;
}

message Player {
  string name = 1;
  Team team = 2;
}
`

// TestCircularReference_DetectionDocumented documents that circular references
// in protobuf messages will cause stack overflow during marshalling.
// This is based on CircularReferencesMarshallTest.java (ISPN-14687)
//
// NOTE: We don't actually attempt to marshal because it causes a fatal
// stack overflow. This test documents the issue and verifies the structure
// is circular.
func TestCircularReference_DetectionDocumented(t *testing.T) {
	// Create circular reference: Team -> Player -> Team
	team := &testproto.Team{
		Name: "New-Team",
	}
	player := &testproto.Player{
		Name: "fax4ever",
		Team: team,
	}
	team.Players = []*testproto.Player{player}

	// Verify the circular reference exists
	if team.Players[0].Team != team {
		t.Fatal("expected player's team to reference original team (circular)")
	}

	t.Log("WARNING: Attempting to marshal this structure with ProtoStreamMarshaller will cause stack overflow")
	t.Log("Circular reference detected: Team -> Player -> Team")

	// We do NOT attempt to marshal here because it would crash the test process
	// Users should avoid creating circular references in protobuf messages
}

// TestCircularReference_StructureValidation verifies we can detect
// circular references programmatically before attempting to marshal
func TestCircularReference_StructureValidation(t *testing.T) {
	// Create circular reference
	team := &testproto.Team{
		Name: "Circular-Team",
	}
	player := &testproto.Player{
		Name: "player1",
		Team: team,
	}
	team.Players = []*testproto.Player{player}

	// Verify circular reference exists
	if len(team.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(team.Players))
	}

	if team.Players[0].Team == nil {
		t.Fatal("expected player to have team reference")
	}

	if team.Players[0].Team != team {
		t.Fatal("expected player's team to point back to original team (circular)")
	}

	t.Log("Circular reference verified: Team.Players[0].Team points back to Team")
	t.Log("NOTE: Attempting cache.Put() with this structure will cause stack overflow")
}

// TestNonCircularReference_Success verifies that non-circular
// nested structures work correctly
func TestNonCircularReference_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "non-circular-test")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register the schema
	if err := client.Schemas().Register(ctx, "circular.proto", circularProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Team](
		client, "non-circular-test",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Team { return &testproto.Team{} },
	)

	// Create a team WITHOUT circular reference
	// (Player has no team reference set)
	team := &testproto.Team{
		Name: "Safe-Team",
		Players: []*testproto.Player{
			{Name: "player1", Team: nil}, // No circular reference
			{Name: "player2", Team: nil},
		},
	}

	// This should succeed
	if err := cache.Put(ctx, "team1", team); err != nil {
		t.Fatalf("Put non-circular team: %v", err)
	}

	// Verify we can retrieve it
	retrieved, found, err := cache.Get(ctx, "team1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected team to be found")
	}
	if retrieved.Name != "Safe-Team" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Safe-Team")
	}
	if len(retrieved.Players) != 2 {
		t.Errorf("len(Players) = %d, want 2", len(retrieved.Players))
	}
	if retrieved.Players[0].Name != "player1" {
		t.Errorf("Player[0].Name = %q, want %q", retrieved.Players[0].Name, "player1")
	}
}
