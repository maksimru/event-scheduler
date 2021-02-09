package fsm

import (
	"encoding/json"
	"github.com/hashicorp/raft"
	"github.com/maksimru/event-scheduler/message"
	"github.com/maksimru/event-scheduler/storage"
	"github.com/stretchr/testify/assert"
	"io"
	"reflect"
	"testing"
	"time"
)

func Test_prioritizedFSM_Snapshot(t *testing.T) {
	type fields struct {
		storage *storage.PqStorage
	}
	tests := []struct {
		name     string
		fields   fields
		want     raft.FSMSnapshot
		wantErr  bool
		messages []message.Message
	}{
		{
			name: "Checks fsm snapshot creation",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			want: raft.FSMSnapshot(&fsmSnapshot{dump: []message.Message{
				message.NewMessage("msg1", 1000),
				message.NewMessage("msg5", 1200),
				message.NewMessage("msg4", 2000),
			}}),
			wantErr: false,
			messages: []message.Message{
				message.NewMessage("msg1", 1000),
				message.NewMessage("msg5", 1200),
				message.NewMessage("msg4", 2000),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prioritizedFSM{
				storage: tt.fields.storage,
			}
			for _, msg := range tt.messages {
				b.storage.Enqueue(msg)
			}
			got, err := b.Snapshot()
			if (err != nil) != tt.wantErr {
				t.Errorf("Snapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func bootStagingCluster(nodeId string, fsm *prioritizedFSM, snapshotStore *raft.InmemSnapshotStore) (*raft.Raft, raft.ServerAddress) {
	store := raft.NewInmemStore()
	cacheStore, _ := raft.NewLogCache(128, store)
	raftTransportTcpAddr := raft.NewInmemAddr()
	_, transport := raft.NewInmemTransport(raftTransportTcpAddr)
	raftconfig := raft.DefaultConfig()
	raftconfig.LogLevel = "info"
	raftconfig.LocalID = raft.ServerID(nodeId)
	raftconfig.SnapshotThreshold = 512
	raftServer, err := raft.NewRaft(raftconfig, fsm, cacheStore, store, snapshotStore, transport)
	if err != nil {
		panic("exception during staging cluster boot: " + err.Error())
	}
	return raftServer, raftTransportTcpAddr
}

func Test_prioritizedFSM_Restore(t *testing.T) {
	type fields struct {
		storage *storage.PqStorage
	}
	type args struct {
		rClose io.ReadCloser
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []message.Message
		wantErr bool
	}{
		{
			name: "Checks fsm snapshot restoration",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			want: []message.Message{
				message.NewMessage("msg1", 1000),
				message.NewMessage("msg5", 1200),
				message.NewMessage("msg4", 2000),
			},
			wantErr: false,
		},
	}
	for testID, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := tt.fields.storage
			snapshotStore := raft.NewInmemSnapshotStore()

			f := &prioritizedFSM{
				storage: s,
			}

			nodeId := string(rune(testID))
			cluster, clusterAddr := bootStagingCluster(nodeId, f, snapshotStore)
			defer func() {
				_ = cluster.Shutdown()
			}()

			// boot required cluster
			cluster.BootstrapCluster(raft.Configuration{Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(nodeId),
					Address:  clusterAddr,
				},
			}})

			// wait for election
			time.Sleep(time.Second * 3)

			assert.Equal(t, []message.Message{}, f.storage.Dump())

			for _, msg := range tt.want {
				opPayload := CommandPayload{
					Operation: OperationPush,
					Value:     &msg,
				}
				opPayloadData, _ := json.Marshal(opPayload)
				applyFuture := cluster.Apply(opPayloadData, 500*time.Millisecond)
				if err := applyFuture.Error(); err != nil {
					t.Fatal("failed to persist the data: ", err)
				}
			}

			snapshot := cluster.Snapshot()

			if err := snapshot.Error(); err != nil {
				t.Fatal("failed to take the snapshot: ", err)
			}

			snapshots, err := snapshotStore.List()
			if err != nil {
				t.Fatal("failed to list snapshots: ", err)
			}
			assert.Equal(t, 1, len(snapshots))
			for _, s := range snapshots {
				_, source, err := snapshotStore.Open(s.ID)
				if err != nil {
					t.Fatal("failed to open snapshot: ", err)
				}
				err = f.Restore(source)
				if err != nil {
					t.Fatal("failed to restore the snapshot: ", err)
				}
			}

			got := f.storage.Dump()
			if !reflect.DeepEqual(got, tt.want) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
