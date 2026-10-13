package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"enflame/k8s-device-plugin/internal/logger"
)

func GetKubeClient() (*kubernetes.Clientset, error) {
	kubeConfig := ""
	// Get kubernetes config.
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		logger.Error(err, "build rest config failed")
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logger.Error(err, "init kube client set failed")
		return nil, err
	}
	logger.Info("init client-go client success")
	return clientset, nil
}
