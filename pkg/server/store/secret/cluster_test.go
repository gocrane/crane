package secret

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/gocrane/crane/pkg/server/store"
)

func TestAddCluster(t *testing.T) {
	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	secret := DefaultSecretStore()
	fakeClient := fake.NewSimpleClientset(secret)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	err := clusterStore.AddCluster(context.TODO(), cluster)
	if err != nil {
		t.Fatal(err)
	}
	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotCluster, cluster) {
		t.Fatalf("got %v, want %v", gotCluster, cluster)
	}

}

func TestDeleteCluster(t *testing.T) {
	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	secret := DefaultSecretStore()
	fakeClient := fake.NewSimpleClientset(secret)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	err := clusterStore.AddCluster(context.TODO(), cluster)
	if err != nil {
		t.Fatal(err)
	}
	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotCluster, cluster) {
		t.Fatalf("got %v, want %v", gotCluster, cluster)
	}

	err = clusterStore.DeleteCluster(context.TODO(), cluster.Id)
	if err != nil {
		t.Fatal(err)
	}

	_, err = clusterStore.GetCluster(context.TODO(), cluster.Id)
	if !errors.IsNotFound(err) {
		t.Fatalf("cluster %v should be deleted, but still exists, err: %v", cluster.Id, err)
	}

}

func TestUpdateCluster(t *testing.T) {
	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	secret := DefaultSecretStore()
	fakeClient := fake.NewSimpleClientset(secret)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	err := clusterStore.AddCluster(context.TODO(), cluster)
	if err != nil {
		t.Fatal(err)
	}

	newCluster := &store.Cluster{
		Id:         "cls-xxxxxx",
		Name:       "aaaa",
		CraneUrl:   "http://127.0.0.1:8081",
		GrafanaUrl: "yyyyy",
	}

	err = clusterStore.UpdateCluster(context.TODO(), newCluster)
	if err != nil {
		t.Fatal(err)
	}

	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotCluster, newCluster) {
		t.Fatalf("got %v, want %v", gotCluster, newCluster)
	}

}

func TestGetCluster(t *testing.T) {
	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	secret := DefaultSecretStore()
	fakeClient := fake.NewSimpleClientset(secret)
	clusterStore := NewClusters(&datastore{client: fakeClient})
	err := clusterStore.AddCluster(context.TODO(), cluster)
	if err != nil {
		t.Fatal(err)
	}
	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotCluster, cluster) {
		t.Fatalf("got %v, want %v", gotCluster, cluster)
	}

}

func TestListCluster(t *testing.T) {

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	cluster2 := &store.Cluster{
		Id:         "cls-yyyyyy",
		Name:       "test2",
		CraneUrl:   "http://127.0.0.1:8081",
		GrafanaUrl: "http://127.0.0.1:3000",
	}

	clusterList := &store.ClusterList{
		TotalCount: 2,
		Items:      []*store.Cluster{cluster, cluster2},
	}

	secret := DefaultSecretStore()
	fakeClient := fake.NewSimpleClientset(secret)
	clusterStore := NewClusters(&datastore{client: fakeClient})
	err := clusterStore.AddCluster(context.TODO(), cluster)
	if err != nil {
		t.Fatal(err)
	}
	err = clusterStore.AddCluster(context.TODO(), cluster2)
	if err != nil {
		t.Fatal(err)
	}

	gotClusterList, err := clusterStore.ListClusters(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotClusterList, clusterList) {
		t.Fatalf("got %v, want %v", gotClusterList, clusterList)
	}

}
