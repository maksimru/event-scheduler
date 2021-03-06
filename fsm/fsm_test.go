package fsm

import (
	"encoding/json"
	"github.com/hashicorp/raft"
	"github.com/maksimru/event-scheduler/channel"
	pubsublistenerconfig "github.com/maksimru/event-scheduler/listener/pubsub/config"
	"github.com/maksimru/event-scheduler/message"
	"github.com/maksimru/event-scheduler/nodenameresolver"
	pubsubpublisherconfig "github.com/maksimru/event-scheduler/publisher/pubsub/config"
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
		channel  channel.Channel
	}{
		{
			name: "Checks fsm snapshot creation",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			want: raft.FSMSnapshot(&fsmSnapshot{
				messagesDump: map[string][]message.Message{
					"id1": {
						message.NewMessage("msg1", 1000),
						message.NewMessage("msg5", 1200),
						message.NewMessage("msg4", 2000),
					},
				},
				channelsDump: []channel.Channel{
					{
						ID:          "id1",
						Source:      channel.Source{},
						Destination: channel.Destination{},
					},
				},
			}),
			wantErr: false,
			messages: []message.Message{
				message.NewMessage("msg1", 1000),
				message.NewMessage("msg5", 1200),
				message.NewMessage("msg4", 2000),
			},
			channel: channel.Channel{
				ID:          "id1",
				Source:      channel.Source{},
				Destination: channel.Destination{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := prioritizedFSM{
				storage: tt.fields.storage,
			}
			c, _ := b.storage.AddChannel(tt.channel)
			for _, msg := range tt.messages {
				channelStorage, _ := b.storage.GetChannelStorage(c.ID)
				channelStorage.Enqueue(msg)
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

func inmemConfig() *raft.Config {
	conf := raft.DefaultConfig()
	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = 5 * time.Millisecond
	return conf
}

func bootStagingCluster(fsm *prioritizedFSM, snapshotStore *raft.InmemSnapshotStore) (*raft.Raft, *raft.InmemTransport) {
	store := raft.NewInmemStore()
	cacheStore, _ := raft.NewLogCache(128, store)
	_, transport := raft.NewInmemTransport("")
	raftconfig := inmemConfig()
	raftconfig.LogLevel = "info"
	raftconfig.LocalID = nodenameresolver.Resolve(string(transport.LocalAddr()))
	raftconfig.SnapshotThreshold = 512
	raftServer, err := raft.NewRaft(raftconfig, fsm, cacheStore, store, snapshotStore, transport)
	if err != nil {
		panic("exception during staging cluster boot: " + err.Error())
	}
	return raftServer, transport
}

func Test_prioritizedFSM_Restore(t *testing.T) {
	type fields struct {
		storage *storage.PqStorage
	}
	type args struct {
		rClose io.ReadCloser
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		messages map[string][]message.Message
		wantErr  bool
		channels []channel.Channel
	}{
		{
			name: "Checks fsm snapshot restoration in single channel",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			messages: map[string][]message.Message{
				"id1": {
					message.NewMessage("msg1", 1000),
					message.NewMessage("msg5", 1200),
					message.NewMessage("msg4", 2000),
				},
			},
			channels: []channel.Channel{
				{
					ID:          "id1",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
			},
			wantErr: false,
		},
		{
			name: "Checks fsm snapshot restoration in single channel, full config",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			messages: map[string][]message.Message{
				"id1": {
					message.NewMessage("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", 1000),
					message.NewMessage("YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY", 1200),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
					message.NewMessage("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", 2000),
				},
			},
			channels: []channel.Channel{
				{
					ID: "id1",
					Source: channel.Source{
						Driver: "pubsub",
						Config: pubsublistenerconfig.SourceConfig{
							ProjectID:      "project",
							SubscriptionID: "subscription",
							KeyFile:        "key",
						},
					},
					Destination: channel.Destination{
						Driver: "pubsub",
						Config: pubsubpublisherconfig.DestinationConfig{
							ProjectID: "project",
							TopicID:   "topic",
							KeyFile:   "key",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Checks fsm snapshot restoration in single channel, empty channel",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			messages: map[string][]message.Message{
				"id1": {},
			},
			channels: []channel.Channel{
				{
					ID:          "id1",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
			},
			wantErr: false,
		},
		{
			name: "Checks fsm snapshot restoration in multiple channels",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			messages: map[string][]message.Message{
				"id1": {
					message.NewMessage("msg1", 1000),
					message.NewMessage("msg5", 1200),
					message.NewMessage("msg4", 2000),
				},
				"id2": {
					message.NewMessage("msg7", 3000),
					message.NewMessage("msg8", 3200),
					message.NewMessage("msg6", 1000),
				},
			},
			channels: []channel.Channel{
				{
					ID:          "id1",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
				{
					ID:          "id2",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
			},
			wantErr: false,
		},
		{
			name: "Checks fsm snapshot restoration in multiple channels, empty channel",
			fields: fields{
				storage: storage.NewPqStorage(),
			},
			messages: map[string][]message.Message{
				"id1": {},
				"id2": {},
			},
			channels: []channel.Channel{
				{
					ID:          "id1",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
				{
					ID:          "id2",
					Source:      channel.Source{},
					Destination: channel.Destination{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := tt.fields.storage
			snapshotStore := raft.NewInmemSnapshotStore()

			f := &prioritizedFSM{
				storage: s,
			}

			cluster, clusterTransport := bootStagingCluster(f, snapshotStore)
			defer func() {
				_ = cluster.Shutdown()
			}()

			// boot required cluster
			cluster.BootstrapCluster(raft.Configuration{Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       nodenameresolver.Resolve(string(clusterTransport.LocalAddr())),
					Address:  clusterTransport.LocalAddr(),
				},
			}})

			// wait for election
			time.Sleep(time.Second * 1)

			gotChannels, gotMessages := f.storage.Dump()
			if !reflect.DeepEqual(gotChannels, tt.channels) {
				assert.Equal(t, []channel.Channel{}, gotChannels)
			}
			if !reflect.DeepEqual(gotMessages, tt.messages) {
				assert.Equal(t, map[string][]message.Message{}, gotMessages)
			}

			for _, c := range tt.channels {
				opPayload := CommandPayload{
					Operation: OperationChannelCreate,
					Channel:   c,
				}
				opPayloadData, _ := json.Marshal(opPayload)
				applyFuture := cluster.Apply(opPayloadData, 500*time.Millisecond)
				if err := applyFuture.Error(); err != nil {
					t.Fatal("failed to persist the data: ", err)
				}
				r, ok := applyFuture.Response().(*ApplyResponse)
				if !ok {
					t.Fatal("error parsing apply response")
				}
				appliedChannel := r.Data.(channel.Channel)

				for _, msg := range tt.messages[appliedChannel.ID] {
					opPayload := CommandPayload{
						Operation: OperationMessagePush,
						ChannelID: appliedChannel.ID,
						Message:   msg,
					}
					opPayloadData, _ := json.Marshal(opPayload)
					applyFuture := cluster.Apply(opPayloadData, 500*time.Millisecond)
					if err := applyFuture.Error(); err != nil {
						t.Fatal("failed to persist the data: ", err)
					}
				}
			}

			snapshot := cluster.Snapshot()

			if err := snapshot.Error(); err != nil {
				assert.NoError(t, err)
				t.Fatal("failed to take the snapshot: ", err)
			}

			snapshots, err := snapshotStore.List()
			assert.NoError(t, err)
			if err != nil {
				t.Fatal("failed to list snapshots: ", err)
			}
			assert.Equal(t, 1, len(snapshots))

			// ensure storage is empty
			f.storage.Flush()

			for _, s := range snapshots {
				_, source, err := snapshotStore.Open(s.ID)
				assert.NoError(t, err)
				if err != nil {
					t.Fatal("failed to open snapshot: ", err)
				}
				err = f.Restore(source)
				assert.NoError(t, err)
				if err != nil {
					t.Fatal("failed to restore the snapshot: ", err)
				}
			}

			gotChannels, gotMessages = f.storage.Dump()
			if !reflect.DeepEqual(gotChannels, tt.channels) {
				assert.ElementsMatch(t, tt.channels, gotChannels)
			}
			for k, msgs := range tt.messages {
				assert.Equal(t, len(msgs), len(gotMessages[k]))
				assert.Equal(t, msgs, gotMessages[k])
			}
		})
	}
}
