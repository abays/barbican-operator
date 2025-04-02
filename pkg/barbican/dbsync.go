package barbican

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"

	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DBSyncCommand -
	DBSyncCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"
)

// DbSyncJob func
func DbSyncJob(instance *barbicanv1beta1.Barbican, labels map[string]string, annotations map[string]string) *batchv1.Job {
	// The dbsync job just needs the main barbican config files
	dbSyncVolumes := []corev1.Volume{}
	dbSyncVolumes = append(dbSyncVolumes, GetVolumes(instance.Name)...)

	dbSyncMounts := []corev1.VolumeMount{
		GetKollaConfigVolumeMount(instance.Name + "-dbsync"),
	}
	dbSyncMounts = append(dbSyncMounts, GetVolumeMounts()...)

	// add CA cert if defined
	if instance.Spec.BarbicanAPI.TLS.CaBundleSecretName != "" {
		dbSyncVolumes = append(dbSyncVolumes, instance.Spec.BarbicanAPI.TLS.CreateVolume())
		dbSyncMounts = append(dbSyncMounts, instance.Spec.BarbicanAPI.TLS.CreateVolumeMounts(nil)...)
	}

	args := []string{"-c", DBSyncCommand}

	runAsUser := int64(0)
	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("TRUE")

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-db-sync",
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: instance.RbacResourceName(),
					Volumes:            dbSyncVolumes,
					Containers: []corev1.Container{
						{
							Name: instance.Name + "-db-sync",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.BarbicanAPI.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: dbSyncMounts,
						},
					},
				},
			},
		},
	}

	if instance.Spec.NodeSelector != nil {
		job.Spec.Template.Spec.NodeSelector = *instance.Spec.NodeSelector
	}

	return job
}
