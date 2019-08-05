package statefulsets

import (
	"context"
	brokerv2alpha1 "github.com/rh-messaging/activemq-artemis-operator/pkg/apis/broker/v2alpha1"
	pvc "github.com/rh-messaging/activemq-artemis-operator/pkg/resources/persistentvolumeclaims"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/utils/selectors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("package statefulsets")

const (
	graceTime       = 30
	TCPLivenessPort = 8161
)

func makeVolumeMounts(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.VolumeMount {

	volumeMounts := []corev1.VolumeMount{}
	if cr.Spec.DeploymentPlan.PersistenceEnabled {
		persistentCRVlMnt := makePersistentVolumeMount(cr)
		volumeMounts = append(volumeMounts, persistentCRVlMnt...)
	}
	if checkSSLEnabled(cr) {
		sslCRVlMnt := makeSSLVolumeMount(cr)
		volumeMounts = append(volumeMounts, sslCRVlMnt...)
	}
	return volumeMounts

}

func makeVolumes(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.Volume {

	volume := []corev1.Volume{}
	if cr.Spec.DeploymentPlan.PersistenceEnabled {
		basicCRVolume := makePersistentVolume(cr)
		volume = append(volume, basicCRVolume...)
	}
	if checkSSLEnabled(cr) {
		sslCRVolume := makeSSLSecretVolume(cr)
		volume = append(volume, sslCRVolume...)
	}
	return volume
}

func makePersistentVolume(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.Volume {

	volume := []corev1.Volume{
		{
			Name: cr.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: cr.Name,
					ReadOnly:  false,
				},
			},
		},
	}

	return volume
}

func makeSSLSecretVolume(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.Volume {

	volume := []corev1.Volume{
		{
			Name: "broker-secret-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "TODO-FIX-REPLACE",//cr.Spec.SSLConfig.SecretName,
				},
			},
		},
	}

	return volume
}

func makePersistentVolumeMount(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.VolumeMount {

	dataPath := getPropertyForCR("AMQ_DATA_DIR", cr, "/opt/"+cr.Name+"/data")
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      cr.Name,
			MountPath: dataPath,
			ReadOnly:  false,
		},
	}
	return volumeMounts
}

func makeSSLVolumeMount(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.VolumeMount {

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "broker-secret-volume",
			MountPath: "/etc/amq-secret-volume",
			ReadOnly:  true,
		},
	}
	return volumeMounts
}

func makeEnvVarArrayForCR(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {

	reqLogger := log.WithName(cr.Name)
	reqLogger.Info("Adding Env varibale ")
	envVar := []corev1.EnvVar{}
	envVarArrayForBasic := addEnvVarForBasic(cr)
	envVar = append(envVar, envVarArrayForBasic...)

	if checkSSLEnabled(cr) {
		envVarArrayForSSL := addEnvVarForSSL(cr)
		envVar = append(envVar, envVarArrayForSSL...)
	}
	if cr.Spec.DeploymentPlan.PersistenceEnabled {
		envVarArrayForPresistent := addEnvVarForPersistent(cr)
		envVar = append(envVar, envVarArrayForPresistent...)
	}
	if checkClusterEnabled(cr) {
		envVarArrayForCluster := addEnvVarForCluster(cr)
		envVar = append(envVar, envVarArrayForCluster...)
	}

	return envVar
}

//func makeEnvVarArrayForCR(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {
//return makeEnvVarArrayForBasicCR(cr)
//}

//TODO: Remove this blatant hack
var GLOBAL_AMQ_CLUSTER_USER string = ""
var GLOBAL_AMQ_CLUSTER_PASSWORD string = ""

func getPropertyForCR(propName string, cr *brokerv2alpha1.ActiveMQArtemis, defaultValue string) string {

	result := defaultValue
	switch propName {
	case "AMQ_USER":
		if len(cr.Spec.DeploymentPlan.User) > 0 {
			result = cr.Spec.DeploymentPlan.User
		}
	case "AMQ_PASSWORD":
		if len(cr.Spec.DeploymentPlan.Password) > 0 {
			result = cr.Spec.DeploymentPlan.Password
		}
	case "AMQ_CLUSTER_USER":
		if len(cr.Spec.DeploymentPlan.ClusterUser) > 0 {
			result = cr.Spec.DeploymentPlan.ClusterUser
			GLOBAL_AMQ_CLUSTER_USER = result
		}
	case "AMQ_CLUSTER_PASSWORD":
		if len(cr.Spec.DeploymentPlan.ClusterPassword) > 0 {
			result = cr.Spec.DeploymentPlan.ClusterPassword
			GLOBAL_AMQ_CLUSTER_PASSWORD = result
		}
	case "AMQ_KEYSTORE_TRUSTSTORE_DIR":
		//if checkSSLEnabled(cr) && len(cr.Spec.SSLConfig.SecretName) > 0 {
		if false { // "TODO-FIX-REPLACE"
			//result = "/etc/amq-secret-volume"
		}
	case "AMQ_TRUSTSTORE":
		//if checkSSLEnabled(cr) && len(cr.Spec.SSLConfig.SecretName) > 0 {
		if false { // "TODO-FIX-REPLACE"
			//result = cr.Spec.SSLConfig.TrustStoreFilename
		}
	case "AMQ_TRUSTSTORE_PASSWORD":
		//if checkSSLEnabled(cr) && len(cr.Spec.SSLConfig.SecretName) > 0 {
		if false { // "TODO-FIX-REPLACE"
			//result = cr.Spec.SSLConfig.TrustStorePassword
		}
	case "AMQ_KEYSTORE":
		//if checkSSLEnabled(cr) && len(cr.Spec.SSLConfig.SecretName) > 0 {
		if false { // "TODO-FIX-REPLACE"
			//result = cr.Spec.SSLConfig.KeystoreFilename
		}
	case "AMQ_KEYSTORE_PASSWORD":
		//if checkSSLEnabled(cr) && len(cr.Spec.SSLConfig.SecretName) > 0 {
		if false { // "TODO-FIX-REPLACE"
			//result = cr.Spec.SSLConfig.KeyStorePassword
		}
	case "AMQ_EXTRA_ARGS":
		if "aio" == strings.ToLower(cr.Spec.DeploymentPlan.JournalType) {
			result = "--aio"
		} else {
			result = "--nio"
		}
	}
	return result
}

func addEnvVarForBasic(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {

	envVarArray := []corev1.EnvVar{
		{
			"AMQ_USER",
			getPropertyForCR("AMQ_USER", cr, "admin"),
			nil,
		},
		{
			"AMQ_PASSWORD",
			getPropertyForCR("AMQ_PASSWORD", cr, "admin"),
			nil,
		},
		{
			"AMQ_ROLE",
			getPropertyForCR("AMQ_ROLE", cr, "admin"),
			nil,
		},
		{
			"AMQ_NAME",
			getPropertyForCR("AMQ_NAME", cr, "amq-broker"),
			nil,
		},
		{
			"AMQ_TRANSPORTS",
			getPropertyForCR("AMQ_TRANSPORTS", cr, "openwire,amqp,stomp,mqtt,hornetq"),
			nil,
		},
		{
			"AMQ_QUEUES",
			getPropertyForCR("AMQ_QUEUES", cr, ""),
			nil,
		},
		{
			"AMQ_ADDRESSES",
			getPropertyForCR("AMQ_ADDRESSES", cr, ""),
			nil,
		},
		{
			"AMQ_GLOBAL_MAX_SIZE",
			getPropertyForCR("AMQ_GLOBAL_MAX_SIZE", cr, "100 mb"),
			nil,
		},
		{
			"AMQ_REQUIRE_LOGIN",
			getPropertyForCR("AMQ_REQUIRE_LOGIN", cr, ""),
			nil,
		},
		{
			"AMQ_EXTRA_ARGS",
			getPropertyForCR("AMQ_EXTRA_ARGS", cr, "--no-autotune"),
			nil,
		},
		{
			"AMQ_ANYCAST_PREFIX",
			getPropertyForCR("AMQ_ANYCAST_PREFIX", cr, ""),
			nil,
		},
		{
			"AMQ_MULTICAST_PREFIX",
			getPropertyForCR("AMQ_MULTICAST_PREFIX", cr, ""),
			nil,
		},
		{
			"POD_NAMESPACE",
			"", // Set to the field metadata.namespace in current object
			nil,
		},
	}

	return envVarArray
}

func addEnvVarForPersistent(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {

	envVarArray := []corev1.EnvVar{
		{
			"AMQ_DATA_DIR",
			getPropertyForCR("AMQ_DATA_DIR", cr, "/opt/"+cr.Name+"/data"),
			nil,
		},
		{
			"AMQ_DATA_DIR_LOGGING",
			getPropertyForCR("AMQ_DATA_DIR_LOGGING", cr, "true"),
			nil,
		},
	}

	return envVarArray
}

func addEnvVarForSSL(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {

	envVarArray := []corev1.EnvVar{
		{
			"AMQ_KEYSTORE_TRUSTSTORE_DIR",
			getPropertyForCR("AMQ_KEYSTORE_TRUSTSTORE_DIR", cr, "/etc/amq-secret-volume"),
			nil,
		},
		{
			"AMQ_TRUSTSTORE",
			getPropertyForCR("AMQ_TRUSTSTORE", cr, ""),
			nil,
		},
		{
			"AMQ_TRUSTSTORE_PASSWORD",
			getPropertyForCR("AMQ_TRUSTSTORE_PASSWORD", cr, ""),
			nil,
		},
		{
			"AMQ_KEYSTORE",
			getPropertyForCR("AMQ_KEYSTORE", cr, ""),
			nil,
		},
		{
			"AMQ_KEYSTORE_PASSWORD",
			getPropertyForCR("AMQ_KEYSTORE_PASSWORD", cr, ""),
			nil,
		},
	}

	return envVarArray

}
func addEnvVarForCluster(cr *brokerv2alpha1.ActiveMQArtemis) []corev1.EnvVar {

	envVarArray := []corev1.EnvVar{

		{
			"AMQ_CLUSTERED",
			getPropertyForCR("AMQ_CLUSTERED", cr, "true"),
			nil,
		},
		{
			"AMQ_CLUSTER_USER",
			getPropertyForCR("AMQ_CLUSTER_USER", cr, "clusteruser"),
			nil,
		},
		{
			"AMQ_CLUSTER_PASSWORD",
			getPropertyForCR("AMQ_CLUSTER_PASSWORD", cr, "clusterpass"),
			nil,
		},
	}

	return envVarArray
}

func newEnvVarArrayForCR(cr *brokerv2alpha1.ActiveMQArtemis) *[]corev1.EnvVar {

	envVarArray := makeEnvVarArrayForCR(cr)

	return &envVarArray
}

func newPodTemplateSpecForCR(cr *brokerv2alpha1.ActiveMQArtemis) corev1.PodTemplateSpec {

	// Log where we are and what we're doing
	reqLogger := log.WithName(cr.Name)
	reqLogger.Info("Creating new pod template spec for custom resource")

	//var pts corev1.PodTemplateSpec
	terminationGracePeriodSeconds := int64(60)
	pts := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    cr.Labels,
		},
	}
	Spec := corev1.PodSpec{}
	Containers := []corev1.Container{}
	container := corev1.Container{
		Name:    cr.Name + "-container",
		Image:   cr.Spec.DeploymentPlan.Image,
		Command: []string{"/opt/amq/bin/launch.sh", "start"},
		Env:     makeEnvVarArrayForCR(cr),
		ReadinessProbe: &corev1.Probe{
			InitialDelaySeconds: graceTime,
			TimeoutSeconds:      5,
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/bash",
						"-c",
						"/opt/amq/bin/readinessProbe.sh",
					},
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			InitialDelaySeconds: graceTime,
			TimeoutSeconds:      5,
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(TCPLivenessPort),
				},
			},
		},
	}
	volumeMounts := makeVolumeMounts(cr)
	if len(volumeMounts) > 0 {
		container.VolumeMounts = volumeMounts
	}
	Spec.Containers = append(Containers, container)
	volumes := makeVolumes(cr)
	if len(volumes) > 0 {
		Spec.Volumes = volumes
	}
	Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds
	pts.Spec = Spec

	return pts
}

func newStatefulSetForCR(cr *brokerv2alpha1.ActiveMQArtemis) *appsv1.StatefulSet {

	// Log where we are and what we're doing
	reqLogger := log.WithName(cr.Name)
	reqLogger.Info("Creating new statefulset for custom resource")
	replicas := cr.Spec.DeploymentPlan.Size

	labels := selectors.LabelsForActiveMQArtemis(cr.Name)

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name + "-ss", //"-statefulset",
			Namespace:   cr.Namespace,
			Labels:      labels,
			Annotations: cr.Annotations,
		},
	}
	Spec := appsv1.StatefulSetSpec{
		Replicas:    &replicas,
		ServiceName: "amq-broker-amq-headless", //cr.Name + "-headless" + "-service",
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: newPodTemplateSpecForCR(cr),
	}

	if cr.Spec.DeploymentPlan.PersistenceEnabled {
		Spec.VolumeClaimTemplates = *pvc.NewPersistentVolumeClaimArrayForCR(cr, 1)
	}
	ss.Spec = Spec

	return ss
}

func checkSSLEnabled(cr *brokerv2alpha1.ActiveMQArtemis) bool {
	reqLogger := log.WithName(cr.Name)
	var sslEnabled = false
	//if len(cr.Spec.SSLConfig.SecretName) != 0 && len(cr.Spec.SSLConfig.KeyStorePassword) != 0 && len(cr.Spec.SSLConfig.KeystoreFilename) != 0 && len(cr.Spec.SSLConfig.TrustStorePassword) != 0 && len(cr.Spec.SSLConfig.TrustStoreFilename) != 0 {
	if false { // "TODO-FIX-REPLACE"
		reqLogger.Info("SSL enabled and SSLConfig Section Provided")
		sslEnabled = true
	}
	return sslEnabled
}
func checkClusterEnabled(cr *brokerv2alpha1.ActiveMQArtemis) bool {
	reqLogger := log.WithName(cr.Name)
	var clusterEnabled = false
	//if len(cr.Spec.DeploymentPlan.ClusterUser) != 0 && len(cr.Spec.DeploymentPlan.ClusterPassword) != 0 {
	if false { // "TODO-FIX-REPLACE"
		reqLogger.Info("clustering enabled ")
		clusterEnabled = true
	}
	return clusterEnabled
}

var GLOBAL_CRNAME string = ""

func CreateStatefulSet(cr *brokerv2alpha1.ActiveMQArtemis, client client.Client, scheme *runtime.Scheme) (*appsv1.StatefulSet, error) {

	// Log where we are and what we're doing
	reqLogger := log.WithValues("ActiveMQArtemis Name", cr.Name)
	reqLogger.Info("Creating new statefulset")
	var err error = nil

	// Define the StatefulSet
	ss := newStatefulSetForCR(cr)

	// Set ActiveMQArtemis instance as the owner and controller
	if err = controllerutil.SetControllerReference(cr, ss, scheme); err != nil {
		// Add error detail for use later
		reqLogger.Info("Failed to set controller reference for new " + "statefulset")
	}
	reqLogger.Info("Set controller reference for new " + "statefulset")

	// Call k8s create for statefulset
	if err = client.Create(context.TODO(), ss); err != nil {
		// Add error detail for use later
		reqLogger.Info("Failed to creating new " + "statefulset")
	}
	reqLogger.Info("Created new " + "statefulset")

	//TODO: Remove this blatant hack
	GLOBAL_CRNAME = cr.Name

	return ss, err
}
func RetrieveStatefulSet(statefulsetName string, namespacedName types.NamespacedName, client client.Client) (*appsv1.StatefulSet, error) {

	// Log where we are and what we're doing
	reqLogger := log.WithValues("ActiveMQArtemis Name", namespacedName.Name)
	reqLogger.Info("Retrieving " + "statefulset")

	var err error = nil

	// TODO: Remove this hack
	var crName string = statefulsetName

	labels := selectors.LabelsForActiveMQArtemis(crName)

	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        statefulsetName,
			Namespace:   namespacedName.Namespace,
			Labels:      labels,
			Annotations: nil,
		},
	}
	// Check if the headless service already exists
	if err = client.Get(context.TODO(), namespacedName, ss); err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Statefulset claim IsNotFound", "Namespace", namespacedName.Namespace, "Name", namespacedName.Name)
		} else {
			reqLogger.Info("Statefulset claim found", "Namespace", namespacedName.Namespace, "Name", namespacedName.Name)
		}
	}

	return ss, err
}
