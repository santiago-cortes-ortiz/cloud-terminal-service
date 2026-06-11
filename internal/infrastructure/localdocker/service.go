package localdocker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"

	appsecr "aws-terminal/internal/application/ecr"
	domainecr "aws-terminal/internal/domain/ecr"
)

type Service struct {
	client       *client.Client
	registryAuth string
}

func NewService() (*Service, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return NewServiceWithClient(cli), nil
}

func NewServiceWithClient(cli *client.Client) *Service { return &Service{client: cli} }

func (s *Service) ListLocalImages(ctx context.Context) ([]domainecr.LocalImage, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("Docker Engine client is not configured")
	}
	imageList, err := s.client.ImageList(ctx, client.ImageListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("list local Docker images: %w", err)
	}
	result := make([]domainecr.LocalImage, 0, len(imageList.Items))
	for _, image := range imageList.Items {
		refs := image.RepoTags
		if len(refs) == 0 {
			refs = []string{image.ID}
		}
		for _, ref := range refs {
			if ref == "<none>:<none>" || strings.TrimSpace(ref) == "" {
				continue
			}
			repo, tag := splitImageReference(ref)
			result = append(result, domainecr.LocalImage{ID: image.ID, Repository: repo, Tag: tag, Reference: ref, SizeBytes: image.Size, CreatedAt: time.Unix(image.Created, 0)})
		}
	}
	return result, nil
}

func (s *Service) Login(ctx context.Context, auth domainecr.AuthorizationToken) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("Docker Engine client is not configured")
	}
	authConfig := registry.AuthConfig{Username: auth.Username, Password: auth.Password, ServerAddress: strings.TrimPrefix(strings.TrimPrefix(auth.ProxyEndpoint, "https://"), "http://")}
	encoded, err := encodeAuthConfig(authConfig)
	if err != nil {
		return fmt.Errorf("encode Docker registry auth: %w", err)
	}
	_, err = s.client.RegistryLogin(ctx, client.RegistryLoginOptions{Username: authConfig.Username, Password: authConfig.Password, ServerAddress: authConfig.ServerAddress})
	if err != nil {
		return fmt.Errorf("Docker login to ECR: %w", err)
	}
	s.registryAuth = encoded
	return nil
}

func (s *Service) TagImage(ctx context.Context, sourceImage, destinationImage string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("Docker Engine client is not configured")
	}
	if _, err := s.client.ImageTag(ctx, client.ImageTagOptions{Source: strings.TrimSpace(sourceImage), Target: strings.TrimSpace(destinationImage)}); err != nil {
		return fmt.Errorf("tag Docker image: %w", err)
	}
	return nil
}

func (s *Service) PushImage(ctx context.Context, destinationImage string, progress chan<- domainecr.PushProgress) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("Docker Engine client is not configured")
	}
	encodedAuth := s.registryAuth
	if encodedAuth == "" {
		var err error
		encodedAuth, err = encodeAuthConfig(registry.AuthConfig{})
		if err != nil {
			return "", err
		}
	}
	reader, err := s.client.ImagePush(ctx, strings.TrimSpace(destinationImage), client.ImagePushOptions{RegistryAuth: encodedAuth})
	if err != nil {
		return "", fmt.Errorf("push Docker image: %w", err)
	}
	defer reader.Close()
	return decodePushStream(reader, progress)
}

var _ appsecr.DockerAPI = (*Service)(nil)

func encodeAuthConfig(auth registry.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

func splitImageReference(ref string) (string, string) {
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		return ref[:lastColon], ref[lastColon+1:]
	}
	return ref, "latest"
}

func decodePushStream(reader io.Reader, progress chan<- domainecr.PushProgress) (string, error) {
	decoder := json.NewDecoder(reader)
	digest := ""
	for decoder.More() {
		var message jsonstream.Message
		if err := decoder.Decode(&message); err != nil {
			return digest, err
		}
		if message.Aux != nil {
			var aux struct {
				Digest string `json:"Digest"`
			}
			_ = json.Unmarshal(*message.Aux, &aux)
			if aux.Digest != "" {
				digest = aux.Digest
			}
		}
		if message.Error != nil {
			if progress != nil {
				progress <- domainecr.PushProgress{Status: message.Status, ID: message.ID, Error: message.Error.Message}
			}
			return digest, fmt.Errorf("%s", message.Error.Message)
		}
		if progress != nil {
			p := domainecr.PushProgress{Status: message.Status, ID: message.ID}
			if message.Progress != nil {
				p.Current = message.Progress.Current
				p.Total = message.Progress.Total
				if p.Total > 0 {
					p.Detail = fmt.Sprintf("%d/%d", p.Current, p.Total)
				}
			}
			progress <- p
		}
	}
	return digest, nil
}
