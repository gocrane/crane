package configmap

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

	fakeClient := fake.NewSimpleClientset()
	clusterStore := NewClusters(&datastore{client: fakeClient})

	err := clusterStore.AddCluster(context.TODO(), cluster, &store.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id, &store.GetOptions{})
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

	cmCluster, err := Cluster2ConfigMap(cluster)
	if err != nil {
		t.Fatal(err)
	}
	fakeClient := fake.NewSimpleClientset(cmCluster)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id, &store.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotCluster, cluster) {
		t.Fatalf("got %v, want %v", gotCluster, cluster)
	}

	err = clusterStore.DeleteCluster(context.TODO(), cluster.Id, &store.DeleteOptions{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clusterStore.GetCluster(context.TODO(), cluster.Id, &store.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Fatalf("cluster %v should deleted, but still exists", cluster.Id)
	}

}

func TestUpdateCluster(t *testing.T) {
	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	cmCluster, err := Cluster2ConfigMap(cluster)
	if err != nil {
		t.Fatal(err)
	}
	fakeClient := fake.NewSimpleClientset(cmCluster)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	newCluster := &store.Cluster{
		Id:         "cls-xxxxxx",
		Name:       "aaaa",
		CraneUrl:   "http://127.0.0.1:8081",
		GrafanaUrl: "yyyyy",
	}

	err = clusterStore.UpdateCluster(context.TODO(), newCluster, &store.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id, &store.GetOptions{})
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

	cmCluster, err := Cluster2ConfigMap(cluster)
	if err != nil {
		t.Fatal(err)
	}
	fakeClient := fake.NewSimpleClientset(cmCluster)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	gotCluster, err := clusterStore.GetCluster(context.TODO(), cluster.Id, &store.GetOptions{})
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

	clusterList := &store.ClusterList{
		TotalCount: 1,
		Items:      []*store.Cluster{cluster},
	}

	cmCluster, err := Cluster2ConfigMap(cluster)
	if err != nil {
		t.Fatal(err)
	}
	fakeClient := fake.NewSimpleClientset(cmCluster)
	clusterStore := NewClusters(&datastore{client: fakeClient})

	gotClusterList, err := clusterStore.ListClusters(context.TODO(), &store.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotClusterList, clusterList) {
		t.Fatalf("got %v, want %v", gotClusterList, clusterList)
	}

}
