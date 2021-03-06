// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"context"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"path"

	"github.com/ystia/yorc/v4/tosca"
)

// StoreRepositories store repositories
func StoreRepositories(ctx context.Context, topology tosca.Topology, topologyPrefix string) error {
	repositoriesPrefix := path.Join(topologyPrefix, "repositories")
	kv := make([]*types.KeyValue, 0)
	for repositoryName, repo := range topology.Repositories {
		repoPrefix := path.Join(repositoriesPrefix, repositoryName)
		// Default repository token is password
		if repo.Credit.TokenType == "" {
			repo.Credit.TokenType = "password"
		}
		kv = append(kv, &types.KeyValue{
			Key:   repoPrefix,
			Value: repo,
		})
	}
	return storage.GetStore(types.StoreTypeDeployment).SetCollection(ctx, kv)
}
