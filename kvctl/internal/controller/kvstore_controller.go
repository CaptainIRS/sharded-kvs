/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"log"
	"maps"
	"math/rand"
	"slices"
	"strconv"
	"time"

	pb "github.com/CaptainIRS/sharded-kvs/internal/protos"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kvctlv1 "github.com/CaptainIRS/sharded-kvs/api/v1"
)

// KVStoreReconciler reconciles a KVStore object
type KVStoreReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type KVStoreShard struct {
	Name     string   `yaml:"name"`
	Replicas []string `yaml:"replicas"`
}

type KVStoreConfig struct {
	Shards    []KVStoreShard `yaml:"shards"`
	NewShards []KVStoreShard `yaml:"newShards,omitempty"`
}

// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kvctl.captainirs.dev,resources=kvstores/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *KVStoreReconciler) GetShards(ctx context.Context, kvStore kvctlv1.KVStore) []appsv1.StatefulSet {
	log := logf.FromContext(ctx)
	var shardList appsv1.StatefulSetList
	if err := r.List(ctx, &shardList, client.InNamespace(kvStore.Namespace), client.MatchingFields{shardOwnerKey: kvStore.Name}); err != nil {
		log.Error(err, "unable to fetch shard list")
		return make([]appsv1.StatefulSet, 0)
	}
	return shardList.Items
}

func (r *KVStoreReconciler) GetServices(ctx context.Context, kvStore kvctlv1.KVStore) []corev1.Service {
	log := logf.FromContext(ctx)
	var serviceList corev1.ServiceList
	if err := r.List(ctx, &serviceList, client.InNamespace(kvStore.Namespace), client.MatchingFields{shardOwnerKey: kvStore.Name}); err != nil {
		log.Error(err, "unable to fetch service list")
		return make([]corev1.Service, 0)
	}
	log.Info("Current serviceList", "serviceList", serviceList)
	return serviceList.Items
}

func (r *KVStoreReconciler) GetIngresses(ctx context.Context, kvStore kvctlv1.KVStore) []networkingv1.Ingress {
	log := logf.FromContext(ctx)
	var ingressList networkingv1.IngressList
	if err := r.List(ctx, &ingressList, client.InNamespace(kvStore.Namespace), client.MatchingFields{shardOwnerKey: kvStore.Name}); err != nil {
		log.Error(err, "unable to fetch ingress list")
		return make([]networkingv1.Ingress, 0)
	}
	log.Info("Current ingressList", "ingressList", ingressList)
	return ingressList.Items
}

func (r *KVStoreReconciler) UpdateConfigMap(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	name := kvStore.Name

	kvConfig := KVStoreConfig{}
	kvConfig.Shards = make([]KVStoreShard, 0)
	for s := 0; s < int(kvStore.Status.CurrentShards); s++ {
		shardConfig := KVStoreShard{}
		shardConfig.Name = fmt.Sprintf("%s-shard-%d", name, s)
		shardReplicas := make([]string, 0)
		for r := 0; r < int(kvStore.Spec.Replicas); r++ {
			shardReplicas = append(shardReplicas, fmt.Sprintf("%s-replica-%d", shardConfig.Name, r))
		}
		shardConfig.Replicas = shardReplicas
		kvConfig.Shards = append(kvConfig.Shards, shardConfig)
	}
	if kvStore.Status.CurrentShards != kvStore.Status.TargetShards || kvStore.Status.CurrentShards == 0 {
		for s := 0; s < int(kvStore.Status.TargetShards); s++ {
			shardConfig := KVStoreShard{}
			shardConfig.Name = fmt.Sprintf("%s-shard-%d", name, s)
			shardReplicas := make([]string, 0)
			for r := 0; r < int(kvStore.Spec.Replicas); r++ {
				shardReplicas = append(shardReplicas, fmt.Sprintf("%s-replica-%d", shardConfig.Name, r))
			}
			shardConfig.Replicas = shardReplicas
			if kvStore.Status.CurrentShards == 0 {
				kvConfig.Shards = append(kvConfig.Shards, shardConfig)
			} else {
				kvConfig.NewShards = append(kvConfig.NewShards, shardConfig)
			}
		}
	}
	configYaml := ""
	if yamlBytes, err := yaml.Marshal(kvConfig); err != nil {
		log.Error(err, "could not build shard configuration YAML")
		return err
	} else {
		configYaml = string(yamlBytes)
	}

	var existingConfigMap corev1.ConfigMap

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: kvStore.Namespace,
		Name:      fmt.Sprintf("%s-config", name),
	}, &existingConfigMap); err != nil {
		if errors.IsNotFound(err) {
			configMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-config", name),
					Labels: map[string]string{
						"app": kvStore.Name,
					},
					Namespace: kvStore.Namespace,
				},
				Data: map[string]string{
					"config.yaml": configYaml,
				},
			}
			if err := ctrl.SetControllerReference(&kvStore, &configMap, r.Scheme); err != nil {
				log.Error(err, "failed to set controller reference for configmap", "configMap", configMap)
				return err
			}
			if err := r.Create(ctx, &configMap); err != nil {
				log.Error(err, "failed to create ConfigMap")
				return err
			}
			log.Info("Created ConfigMap successfully")
			return nil
		} else {
			log.Error(err, "failed to create ConfigMap")
			return err
		}
	}
	updatedConfigMap := existingConfigMap.DeepCopy()
	updatedConfigMap.Data = map[string]string{
		"config.yaml": configYaml,
	}
	if err := r.Update(ctx, updatedConfigMap); err != nil {
		log.Error(err, "failed to update ConfigMap")
		return err
	}
	log.Info("Successfully updated ConfigMap")
	return nil
}

func (r *KVStoreReconciler) SpinUpShards(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	name := kvStore.Name
	for i := kvStore.Status.CurrentShards; i < kvStore.Status.TargetShards; i++ {
		shard := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":   kvStore.Name,
					"group": fmt.Sprintf("%s-shard-%d", name, i),
				},
				Annotations: make(map[string]string),
				Name:        fmt.Sprintf("%s-shard-%d-replica", name, i),
				Namespace:   kvStore.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:    &kvStore.Spec.Replicas,
				ServiceName: fmt.Sprintf("%s-shard-%d", name, i),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":   kvStore.Name,
						"group": fmt.Sprintf("%s-shard-%d", name, i),
					},
				},
				MinReadySeconds: 10,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":   kvStore.Name,
							"group": fmt.Sprintf("%s-shard-%d", name, i),
						},
						Annotations: map[string]string{
							"replicas": strconv.Itoa(int(kvStore.Spec.Replicas)),
						},
					},
					Spec: *kvStore.Spec.PodSpec.DeepCopy(),
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data-shard",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Mi"),
							},
						},
					},
				}},
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
					WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
			},
		}
		maps.Copy(shard.Labels, kvStore.Labels)
		if err := ctrl.SetControllerReference(&kvStore, shard, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference for shard", "shard", shard)
			return err
		}
		if err := r.Create(ctx, shard); err != nil {
			log.Error(err, "failed to create shard", "shard", shard)
			return err
		}
		log.Info(fmt.Sprintf("created %s-shard-%d", name, i), "shard", shard)
	}
	return nil
}

func (r *KVStoreReconciler) SpinUpShardIngresses(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	name := kvStore.Name

	for i := kvStore.Status.CurrentShards; i < kvStore.Status.TargetShards; i++ {
		nginx := "nginx"
		pathType := networkingv1.PathTypePrefix
		shardIngress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/ssl-redirect":     "false",
					"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
					"nginx.ingress.kubernetes.io/upstream-vhost":   fmt.Sprintf("%s-shard-%d", name, i),
				},
				Name: fmt.Sprintf("%s-shard-%d-ingress", name, i),
				Labels: map[string]string{
					"app":   kvStore.Name,
					"group": fmt.Sprintf("%s-shard-%d", name, i),
				},
				Namespace: kvStore.Namespace,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &nginx,
				Rules: []networkingv1.IngressRule{{
					Host: fmt.Sprintf("%s-shard-%d.%s", name, i, kvStore.Spec.IngressDomain),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: fmt.Sprintf("%s-shard-%d", name, i),
										Port: networkingv1.ServiceBackendPort{
											Number: 8080,
										},
									},
								},
							}},
						},
					},
				}},
			},
		}
		for k, v := range kvStore.Labels {
			shardIngress.Labels[k] = v
		}
		if err := ctrl.SetControllerReference(&kvStore, shardIngress, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference for shard ingress", "shardIngress", shardIngress)
			return err
		}
		if err := r.Create(ctx, shardIngress); err != nil {
			log.Error(err, "failed to create shard ingress", "shardIngress", shardIngress)
			return err
		}
		log.Info(fmt.Sprintf("created %s-shard-%d-ingress", name, i), "shardIngress", shardIngress)
	}
	return nil
}

func (r *KVStoreReconciler) SpinUpShardServices(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	name := kvStore.Name

	for i := kvStore.Status.CurrentShards; i < kvStore.Status.TargetShards; i++ {
		shardService := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-shard-%d", name, i),
				Labels: map[string]string{
					"app":   name,
					"group": fmt.Sprintf("%s-shard-%d", name, i),
				},
				Namespace: kvStore.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app":   name,
					"group": fmt.Sprintf("%s-shard-%d", name, i),
				},
				ClusterIP:                corev1.ClusterIPNone,
				PublishNotReadyAddresses: true,
				Ports: []corev1.ServicePort{
					{
						Name:       "kvrpc",
						Protocol:   corev1.ProtocolTCP,
						Port:       8080,
						TargetPort: intstr.FromInt(8080),
					},
					{
						Name:       "shardrpc",
						Protocol:   corev1.ProtocolTCP,
						Port:       8081,
						TargetPort: intstr.FromInt(8081),
					},
					{
						Name:       "raftrpc",
						Protocol:   corev1.ProtocolTCP,
						Port:       8082,
						TargetPort: intstr.FromInt(8082),
					},
					{
						Name:       "leaderrpc",
						Protocol:   corev1.ProtocolTCP,
						Port:       8083,
						TargetPort: intstr.FromInt(8083),
					},
				},
			},
		}
		maps.Copy(shardService.Labels, kvStore.Labels)
		if err := ctrl.SetControllerReference(&kvStore, &shardService, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference for shard ingress", "shardService", shardService)
			return err
		}
		if err := r.Create(ctx, &shardService); err != nil {
			log.Error(err, "failed to create shard service", "shardService", shardService)
			return err
		}
	}
	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KVStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *KVStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var kvStore kvctlv1.KVStore
	if err := r.Get(ctx, req.NamespacedName, &kvStore); err != nil {
		log.Error(err, "unable to fetch KVStore")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var shardList appsv1.StatefulSetList
	if err := r.List(ctx, &shardList, client.InNamespace(kvStore.Namespace), client.MatchingFields{shardOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to fetch shard list")
		return ctrl.Result{}, err
	}

	for _, shard := range shardList.Items {
		if *shard.Spec.Replicas != kvStore.Spec.Replicas {
			if shard.Spec.Template.ObjectMeta.Annotations == nil {
				shard.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			shard.Spec.Replicas = &kvStore.Spec.Replicas
			shard.Spec.Template.ObjectMeta.Annotations["replicas"] = strconv.Itoa(int(*shard.Spec.Replicas))
			if err := r.Update(ctx, &shard); err != nil {
				return ctrl.Result{}, err
			} else {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
			}
		}
	}

	if kvStore.Status.Phase == kvctlv1.PhaseNormal {
		if len(shardList.Items) < int(kvStore.Spec.Shards) {
			kvStore.Status.Phase = kvctlv1.PhasePreparingScale
			kvStore.Status.Message = "Spinning up shards"
			kvStore.Status.CurrentShards = len(shardList.Items)
			kvStore.Status.TargetShards = int(kvStore.Spec.Shards)
			kvStore.Status.LastTransition = metav1.Now()
			r.Status().Update(ctx, &kvStore)
			if err := r.UpdateConfigMap(ctx, kvStore); err != nil {
				log.Error(err, "failed updating configmap")
				return ctrl.Result{}, err
			}
			if err := r.SpinUpShardIngresses(ctx, kvStore); err != nil {
				log.Error(err, "failed spinning up shard ingresses")
				return ctrl.Result{}, err
			}
			if err := r.SpinUpShardServices(ctx, kvStore); err != nil {
				log.Error(err, "failed spinning up shard services")
				return ctrl.Result{}, err
			}
			if err := r.SpinUpShards(ctx, kvStore); err != nil {
				log.Error(err, "failed spinning up shards")
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		} else if len(shardList.Items) > int(kvStore.Spec.Shards) {
			kvStore.Status.Phase = kvctlv1.PhasePreparingScale
			kvStore.Status.Message = "Spinning up shards"
			kvStore.Status.CurrentShards = len(shardList.Items)
			kvStore.Status.TargetShards = int(kvStore.Spec.Shards)
			kvStore.Status.LastTransition = metav1.Now()
			r.Status().Update(ctx, &kvStore)
			if err := r.UpdateConfigMap(ctx, kvStore); err != nil {
				log.Error(err, "failed updating configmap")
				return ctrl.Result{}, err
			}
		}
	}

	var err error = nil
	switch kvStore.Status.Phase {
	case kvctlv1.PhasePreparingScale:
		err = r.PrepareScale(ctx, kvStore)
	case kvctlv1.PhaseStoppingWrites:
		err = r.StopWrites(ctx, kvStore)
	case kvctlv1.PhaseRedistributingKeys:
		err = r.RedistributeKeys(ctx, kvStore)
	case kvctlv1.PhaseResumingWrites:
		err = r.ResumeWrites(ctx, kvStore)
	case kvctlv1.PhaseFinalizingScale:
		err = r.FinalizeScale(ctx, kvStore)
	case kvctlv1.PhaseNormal:
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
}

func (r *KVStoreReconciler) PrepareScale(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	shards := r.GetShards(ctx, kvStore)
	for _, shard := range shards {
		log.Info(fmt.Sprintf("%d/%d replicas ready", shard.Status.ReadyReplicas, *shard.Spec.Replicas))
		if shard.Status.ReadyReplicas != *shard.Spec.Replicas {
			kvStore.Status.Message = "Waiting for shards to be ready"
			return r.Status().Update(ctx, &kvStore)
		}
	}
	if kvStore.Status.CurrentShards == 0 {
		kvStore.Status.Phase = kvctlv1.PhaseNormal
		kvStore.Status.Message = "Ready"
		kvStore.Status.CurrentShards = kvStore.Status.TargetShards
		kvStore.Status.LastTransition = metav1.Now()
		return r.Status().Update(ctx, &kvStore)
	}
	kvStore.Status.Phase = kvctlv1.PhaseStoppingWrites
	kvStore.Status.Message = "Starting to disable writes"
	log.Info("Shards are ready. Starting to disable writes")
	kvStore.Status.LastTransition = metav1.Now()
	return r.Status().Update(ctx, &kvStore)
}

func GetShardClient(shard appsv1.StatefulSet) pb.ShardRPCClient {
	shardName := shard.Spec.ServiceName
	for {
		conn, err := grpc.Dial(fmt.Sprintf("%s.%s:%d", shardName, shard.Namespace, 8081), grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect to %s. Retrying...", shardName)
			time.Sleep(1 * time.Second)
			continue
		}
		return pb.NewShardRPCClient(conn)
	}
}
func (r *KVStoreReconciler) StopWrites(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	shards := r.GetShards(ctx, kvStore)
	for _, shard := range shards {
		if res, err := GetShardClient(shard).PauseWrites(ctx, &pb.PauseWritesRequest{}); err != nil {
			kvStore.Status.Message = fmt.Sprintf("Failed to pause writes: %s. Retrying", err)
			r.Status().Update(ctx, &kvStore)
			return err
		} else {
			if !res.IsPaused {
				kvStore.Status.Message = "Waiting for shards to pause writes"
				return r.Status().Update(ctx, &kvStore)
			}
		}
	}
	kvStore.Status.Phase = kvctlv1.PhaseRedistributingKeys
	kvStore.Status.Message = "Starting to redistribute keys"
	log.Info("Writes disabled, starting to redistribute keys")
	kvStore.Status.LastTransition = metav1.Now()
	return r.Status().Update(ctx, &kvStore)
}

func (r *KVStoreReconciler) RedistributeKeys(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	shards := r.GetShards(ctx, kvStore)
	pendingKeys := 0
	for _, shard := range shards {
		if res, err := GetShardClient(shard).RedistributeKeys(ctx, &pb.RedistributeKeysRequest{}); err != nil {
			kvStore.Status.Message = fmt.Sprintf("Failed to initiate key redistribution: %s", err)
			r.Status().Update(ctx, &kvStore)
			return err
		} else {
			pendingKeys += int(res.PendingKeys)
		}
	}
	log.Info(fmt.Sprintf("Pending keys to be redistributed: %d", pendingKeys))
	if pendingKeys == 0 {
		kvStore.Status.Phase = kvctlv1.PhaseFinalizingScale
		kvStore.Status.Message = "Starting to finalize scaling operation"
		log.Info("Keys redistributed. Starting finalization")
		kvStore.Status.LastTransition = metav1.Now()
		return r.Status().Update(ctx, &kvStore)
	} else {
		kvStore.Status.Message = fmt.Sprintf("Pending keys to be redistributed: %d", pendingKeys)
	}
	return r.Status().Update(ctx, &kvStore)
}

func (r *KVStoreReconciler) FinalizeScale(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	if kvStore.Status.TargetShards > kvStore.Status.CurrentShards {
		kvStore.Status.Phase = kvctlv1.PhaseResumingWrites
		kvStore.Status.Message = "Starting to resume writes"
		log.Info("Finalized shard resources. Starting to resume writes")
		kvStore.Status.CurrentShards = kvStore.Status.TargetShards
		kvStore.Status.LastTransition = metav1.Now()
		if err := r.UpdateConfigMap(ctx, kvStore); err != nil {
			return err
		}
		return r.Status().Update(ctx, &kvStore)
	}
	groupsToBeDeleted := make([]string, 0)
	for i := range kvStore.Status.CurrentShards {
		if i >= kvStore.Status.TargetShards {
			groupsToBeDeleted = append(groupsToBeDeleted, fmt.Sprintf("%s-shard-%d", kvStore.Name, i))
		}
	}
	shards := r.GetShards(ctx, kvStore)
	for _, shard := range shards {
		if slices.Contains(groupsToBeDeleted, shard.Labels["group"]) {
			if err := r.Delete(ctx, &shard); err != nil {
				kvStore.Status.Message = fmt.Sprintf("Failed to delete shard in group %s: %s", shard.Labels["group"], err)
				return err
			}
		}
	}
	ingresses := r.GetIngresses(ctx, kvStore)
	for _, ingress := range ingresses {
		if slices.Contains(groupsToBeDeleted, ingress.Labels["group"]) {
			if err := r.Delete(ctx, &ingress); err != nil {
				kvStore.Status.Message = fmt.Sprintf("Failed to delete ingress in group %s: %s", ingress.Labels["group"], err)
				return err
			}
		}
	}
	services := r.GetServices(ctx, kvStore)
	for _, service := range services {
		if slices.Contains(groupsToBeDeleted, service.Labels["group"]) {
			if err := r.Delete(ctx, &service); err != nil {
				kvStore.Status.Message = fmt.Sprintf("Failed to delete service in group %s: %s", service.Labels["group"], err)
				return err
			}
		}
	}

	kvStore.Status.Phase = kvctlv1.PhaseResumingWrites
	kvStore.Status.Message = "Starting to resume writes"
	log.Info("Finalized shard resources. Starting to resume writes")
	kvStore.Status.CurrentShards = kvStore.Status.TargetShards
	kvStore.Status.LastTransition = metav1.Now()
	if err := r.UpdateConfigMap(ctx, kvStore); err != nil {
		return err
	}
	return r.Status().Update(ctx, &kvStore)
}

func (r *KVStoreReconciler) ResumeWrites(ctx context.Context, kvStore kvctlv1.KVStore) error {
	log := logf.FromContext(ctx)
	shards := r.GetShards(ctx, kvStore)
	for _, shard := range shards {
		if res, err := GetShardClient(shard).ResumeWrites(ctx, &pb.ResumeWritesRequest{}); err != nil {
			kvStore.Status.Message = fmt.Sprintf("Failed to resume writes: %s. Retrying", err)
			r.Status().Update(ctx, &kvStore)
			return err
		} else {
			if !res.IsResumed {
				kvStore.Status.Message = "Waiting for shards to resume writes"
				return r.Status().Update(ctx, &kvStore)
			}
		}
	}
	kvStore.Status.Phase = kvctlv1.PhaseNormal
	kvStore.Status.Message = "Ready"
	log.Info("Writes resumed. Scaling operation complete")
	kvStore.Status.CurrentShards = kvStore.Status.TargetShards
	kvStore.Status.LastTransition = metav1.Now()

	return r.Status().Update(ctx, &kvStore)
}

var (
	shardOwnerKey = ".metadata.controller"
	apiGVStr      = kvctlv1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *KVStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	rand.Seed(time.Now().UnixNano())
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.StatefulSet{}, shardOwnerKey, func(rawObj client.Object) []string {
		// grab the StatefulSet object, extract the owner...
		statefulSet := rawObj.(*appsv1.StatefulSet)
		owner := metav1.GetControllerOf(statefulSet)
		if owner == nil {
			return nil
		}
		// ...make sure it's a KVStore...
		if owner.APIVersion != apiGVStr || owner.Kind != "KVStore" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkingv1.Ingress{}, shardOwnerKey, func(rawObj client.Object) []string {
		// grab the Ingress object, extract the owner...
		shardIngress := rawObj.(*networkingv1.Ingress)
		owner := metav1.GetControllerOf(shardIngress)
		if owner == nil {
			return nil
		}
		// ...make sure it's a KVStore...
		if owner.APIVersion != apiGVStr || owner.Kind != "KVStore" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, shardOwnerKey, func(rawObj client.Object) []string {
		// grab the Ingress object, extract the owner...
		shardService := rawObj.(*corev1.Service)
		owner := metav1.GetControllerOf(shardService)
		if owner == nil {
			return nil
		}
		// ...make sure it's a KVStore...
		if owner.APIVersion != apiGVStr || owner.Kind != "KVStore" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kvctlv1.KVStore{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&corev1.ConfigMap{}).
		Named("kvstore").
		Complete(r)
}
