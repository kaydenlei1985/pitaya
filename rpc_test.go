// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package pitaya

import (
	"bytes"
	"context"
	"encoding/gob"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/cluster"
	clustermocks "github.com/topfreegames/pitaya/cluster/mocks"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/router"
	serializemocks "github.com/topfreegames/pitaya/serialize/mocks"
	"github.com/topfreegames/pitaya/service"
)

func TestDoSendRPCNotInitialized(t *testing.T) {
	err := doSendRPC(nil, "", "", nil)
	assert.Equal(t, constants.ErrRPCServerNotInitialized, err)
}

func TestDoSendRPC(t *testing.T) {
	app.server.ID = "myserver"
	app.rpcServer = &cluster.NatsRPCServer{}
	tables := []struct {
		name     string
		routeStr string
		reply    interface{}
		args     []interface{}
		err      error
	}{
		{"bad_reply", "", "badreply", nil, constants.ErrReplyShouldBePtr},
		{"bad_route", "badroute", &someStruct{}, nil, route.ErrInvalidRoute},
		{"no_server_type", "bla.bla", &someStruct{}, nil, constants.ErrNoServerTypeChosenForRPC},
		{"nonsense_rpc", "mytype.bla.bla", &someStruct{}, nil, constants.ErrNonsenseRPC},
		{"success", "bla.bla.bla", &someStruct{}, []interface{}{}, nil},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			ctx := context.Background()
			if table.err == nil {
				packetEncoder := codec.NewPomeloPacketEncoder()
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockSerializer := serializemocks.NewMockSerializer(ctrl)
				mockSD := clustermocks.NewMockServiceDiscovery(ctrl)
				mockRPCClient := clustermocks.NewMockRPCClient(ctrl)
				mockRPCServer := clustermocks.NewMockRPCServer(ctrl)
				messageEncoder := message.NewEncoder(false)
				router := router.New()
				svc := service.NewRemoteService(mockRPCClient, mockRPCServer, mockSD, packetEncoder, mockSerializer, router, messageEncoder, &cluster.Server{})
				assert.NotNil(t, svc)
				remoteService = svc
				app.server.ID = "notmyserver"
				buf := bytes.NewBuffer(nil)
				err := gob.NewEncoder(buf).Encode(&someStruct{A: 1})
				assert.NoError(t, err)
				mockSD.EXPECT().GetServer("myserver").Return(&cluster.Server{}, nil)
				mockRPCClient.EXPECT().Call(ctx, protos.RPCType_User, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&protos.Response{Data: buf.Bytes()}, nil)
			}
			err := doSendRPC(ctx, "myserver", table.routeStr, table.reply, table.args...)
			assert.Equal(t, table.err, err)
		})
	}
}