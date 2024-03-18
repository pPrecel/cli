package registry

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
)

// empty struct that implements the httpstream.Connection interface
type connectionMock struct{}

func (cm *connectionMock) CreateStream(headers http.Header) (httpstream.Stream, error) {
	return nil, nil
}

func (cm *connectionMock) Close() error { return nil }

func (cm *connectionMock) CloseChan() <-chan bool { return make(<-chan bool) }

func (cm *connectionMock) SetIdleTimeout(timeout time.Duration) {}

func (cm *connectionMock) RemoveStreams(streams ...httpstream.Stream) {}

func Test_importImage(t *testing.T) {
	type args struct {
		ctx       context.Context
		imageName string
		opts      ImportOptions
		utils     utils
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr error
	}{
		{
			name: "import image",
			args: args{
				ctx:       context.Background(),
				imageName: "test:image",
				opts: ImportOptions{
					ClusterAPIRestConfig: nil,
					RegistryAuth: &basicAuth{
						username: "username",
						password: "password",
					},
					RegistryPullHost:     "testhost:123",
					RegistryPodName:      "podname",
					RegistryPodNamespace: "podnamespace",
					RegistryPodPort:      "1234",
				},
				utils: utils{
					daemonImage: func(r name.Reference, o ...daemon.Option) (v1.Image, error) {
						require.Equal(t, "index.docker.io/library/test:image", r.Name())
						require.Len(t, o, 1)

						return &fake.FakeImage{}, nil
					},
					portforwardNewDial: func(config *rest.Config, podName, podNamespace string) (httpstream.Connection, error) {
						require.Nil(t, config)
						require.Equal(t, "podname", podName)
						require.Equal(t, "podnamespace", podNamespace)

						return &connectionMock{}, nil
					},
					remoteWrite: func(ref name.Reference, img v1.Image, o ...remote.Option) error {
						require.Equal(t, "testhost:123/test:image", ref.Name())
						require.Equal(t, &fake.FakeImage{}, img)
						require.Len(t, o, 3)

						return nil
					},
				},
			},
			wantErr: nil,
			want:    "testhost:123/test:image",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := importImage(tt.args.ctx, tt.args.imageName, tt.args.opts, tt.args.utils)

			require.Equal(t, tt.wantErr, err)
			require.Equal(t, tt.want, got)
		})
	}
}
