package cluster

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/gocrane/crane/pkg/server/store"
)

func TestAddCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := store.NewMockFactory(ctrl)
	mockClusterStore := store.NewMockClusterStore(ctrl)

	mockFactory.EXPECT().Clusters().AnyTimes().Return(mockClusterStore)

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}
	mockClusterStore.EXPECT().AddCluster(gomock.Any(), gomock.Eq(cluster), gomock.Any()).Return(nil)

	type args struct {
		ctx     context.Context
		cluster *store.Cluster
		opts    *store.CreateOptions
	}
	tests := []struct {
		name    string
		store   store.Factory
		args    args
		wantErr bool
	}{
		{
			"1. default",
			mockFactory,
			args{
				context.TODO(),
				cluster,
				&store.CreateOptions{},
			},
			false,
		},
	}

	for _, tc := range tests {
		clusterSrv := NewManager(mockFactory)
		if err := clusterSrv.AddCluster(tc.args.ctx, tc.args.cluster, tc.args.opts); (err != nil) != tc.wantErr {
			t.Errorf("test case %v error = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestGetCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := store.NewMockFactory(ctrl)
	mockClusterStore := store.NewMockClusterStore(ctrl)

	mockFactory.EXPECT().Clusters().AnyTimes().Return(mockClusterStore)

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	mockClusterStore.EXPECT().GetCluster(gomock.Any(), gomock.Eq(cluster.Id), gomock.Any()).Return(cluster, nil)
	type args struct {
		ctx     context.Context
		cluster string
		opts    *store.GetOptions
	}
	tests := []struct {
		name    string
		store   store.Factory
		args    args
		want    *store.Cluster
		wantErr bool
	}{
		{
			"1. default",
			mockFactory,
			args{
				context.TODO(),
				cluster.Id,
				&store.GetOptions{},
			},
			cluster,
			false,
		},
	}

	for _, tc := range tests {
		clusterSrv := NewManager(mockFactory)
		got, err := clusterSrv.GetCluster(tc.args.ctx, tc.args.cluster, tc.args.opts)

		if (err != nil) != tc.wantErr {
			t.Errorf("test case %v error = %v, wantErr %v", tc.name, err, tc.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("test case %v got = %v, want %v", tc.name, got, tc.want)
			return
		}
	}
}

func TestUpdateCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := store.NewMockFactory(ctrl)
	mockClusterStore := store.NewMockClusterStore(ctrl)

	mockFactory.EXPECT().Clusters().AnyTimes().Return(mockClusterStore)

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}
	mockClusterStore.EXPECT().UpdateCluster(gomock.Any(), gomock.Eq(cluster), gomock.Any()).Return(nil)

	type args struct {
		ctx     context.Context
		cluster *store.Cluster
		opts    *store.UpdateOptions
	}
	tests := []struct {
		name    string
		store   store.Factory
		args    args
		wantErr bool
	}{
		{
			"1. default",
			mockFactory,
			args{
				context.TODO(),
				cluster,
				&store.UpdateOptions{},
			},
			false,
		},
	}

	for _, tc := range tests {
		clusterSrv := NewManager(mockFactory)
		if err := clusterSrv.UpdateCluster(tc.args.ctx, tc.args.cluster, tc.args.opts); (err != nil) != tc.wantErr {
			t.Errorf("test case %v error = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestDeleteCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := store.NewMockFactory(ctrl)
	mockClusterStore := store.NewMockClusterStore(ctrl)

	mockFactory.EXPECT().Clusters().AnyTimes().Return(mockClusterStore)

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	mockClusterStore.EXPECT().DeleteCluster(gomock.Any(), gomock.Eq(cluster.Id), gomock.Any()).Return(nil)
	type args struct {
		ctx     context.Context
		cluster string
		opts    *store.DeleteOptions
	}
	tests := []struct {
		name    string
		store   store.Factory
		args    args
		want    *store.Cluster
		wantErr bool
	}{
		{
			"1. default",
			mockFactory,
			args{
				context.TODO(),
				cluster.Id,
				&store.DeleteOptions{},
			},
			cluster,
			false,
		},
	}

	for _, tc := range tests {
		clusterSrv := NewManager(mockFactory)
		err := clusterSrv.DeleteCluster(tc.args.ctx, tc.args.cluster, tc.args.opts)

		if (err != nil) != tc.wantErr {
			t.Errorf("test case %v error = %v, wantErr %v", tc.name, err, tc.wantErr)
			return
		}
	}
}

func TestListCluster(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := store.NewMockFactory(ctrl)
	mockClusterStore := store.NewMockClusterStore(ctrl)

	mockFactory.EXPECT().Clusters().AnyTimes().Return(mockClusterStore)

	cluster := &store.Cluster{
		Id:       "cls-xxxxxx",
		Name:     "test",
		CraneUrl: "http://127.0.0.1:8081",
	}

	clusterList := &store.ClusterList{
		TotalCount: 1,
		Items:      []*store.Cluster{cluster},
	}

	mockClusterStore.EXPECT().ListClusters(gomock.Any(), gomock.Any()).Return(clusterList, nil)
	type args struct {
		ctx  context.Context
		opts *store.ListOptions
	}
	tests := []struct {
		name    string
		store   store.Factory
		args    args
		want    *store.ClusterList
		wantErr bool
	}{
		{
			"1. default",
			mockFactory,
			args{
				context.TODO(),
				&store.ListOptions{},
			},
			clusterList,
			false,
		},
	}

	for _, tc := range tests {
		clusterSrv := NewManager(mockFactory)
		got, err := clusterSrv.ListClusters(tc.args.ctx, tc.args.opts)

		if (err != nil) != tc.wantErr {
			t.Errorf("test case %v error = %v, wantErr %v", tc.name, err, tc.wantErr)
			return
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("test case %v got = %v, want %v", tc.name, got, tc.want)
			return
		}
	}
}
