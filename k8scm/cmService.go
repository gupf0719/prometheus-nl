package k8scm

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"os"
	"sync"
	"time"
)

//const NS = "kube-system"
var NS string

//var cminterface = k8sclient.GetClientset().CoreV1().ConfigMaps(NS)

var cminterface corev1.ConfigMapInterface
var once sync.Once
var ctx = context.Background()

func getCminterface() corev1.ConfigMapInterface {

	once.Do(func() {
		NS = os.Getenv("NAMESPACE")
		if NS == "" {
			panic("Cannot get env of key[NAMESPACE]")
		}
		cminterface = GetClientset().CoreV1().ConfigMaps(NS)
	})

	return cminterface
}

type CmService struct {
	logger log.Logger
}

//datas: 文件名/文件内容
func (this *CmService) get(cmname string) (map[string]string, error) {
	cm, err := getCminterface().Get(ctx, cmname, metav1.GetOptions{})
	if err != nil {
		level.Error(this.logger).Log("msg", "get cm from k8s failed", "cm", cmname, "err", err)
		return nil, err
	}

	return cm.Data, nil
}

//datas: 文件名/文件内容
func (this *CmService) upd(cmname string, datas map[string]string) (string, error) {

	cm, err := getCminterface().Get(ctx, cmname, metav1.GetOptions{})
	if err != nil {
		level.Error(this.logger).Log("msg", "get cm from k8s failed", "cm", cmname, "err", err)
		return "", err
	}
	oldVersion := cm.ResourceVersion

	//不存在 todo
	//cm = &v1.ConfigMap{}
	//cm.Name = PROM_ALERT_CM
	//cm.BinaryData = make(map[string][]byte)
	//
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	//olddata := cm.BinaryData //BinaryData数据无法刷新到k8s
	//if olddata == nil {
	//	olddata = make(map[string][]byte)
	//}
	for k, v := range datas {
		cm.Data[k] = v //覆盖
	}

	_, err = getCminterface().Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		level.Error(this.logger).Log("msg", "upd configmap failed", "cm", cmname, "err", err)
		return "", err
	}

	return oldVersion, nil
}

//datas: 文件名/文件内容
func (this *CmService) updForDel(cmName string, datas []string) (string, error) {

	cm, err := getCminterface().Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		level.Error(this.logger).Log("msg", "get cm from k8s failed", "cm", cmName, "err", err)
		return "", err
	}
	oldVersion := cm.ResourceVersion

	//不存在 todo
	//cm = &v1.ConfigMap{}
	//cm.Name = PROM_ALERT_CM
	//cm.BinaryData = make(map[string][]byte)
	//
	if cm.Data != nil {
		for _, k := range datas {
			delete(cm.Data, k)
		}
	}

	//olddata := cm.BinaryData //BinaryData数据无法刷新到k8s
	//if olddata == nil {
	//	olddata = make(map[string][]byte)
	//}

	_, err = getCminterface().Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		level.Error(this.logger).Log("msg", "upd configmap failed", "cm", cmName, "err", err)
		return "", err
	}

	return oldVersion, nil
}

func (this *CmService) checkUpdFinished(ctx context.Context, cmname string, oldVersion string, ok chan bool) error {
out:
	for {
		time.Sleep(30 * time.Second)

		select {
		case <-ctx.Done():
			break out
		default:
			cm2, err := getCminterface().Get(ctx, cmname, metav1.GetOptions{})
			if err == nil && cm2.ResourceVersion != oldVersion {
				level.Info(this.logger).Log("msg", "Success for update rules configmap version.", "old", oldVersion, "new", cm2.ResourceVersion)
				ok <- true
				return nil
			}
		}

	}

	ok <- false
	level.Error(this.logger).Log("msg", "Timeout for checking configmap version in 10m", "cm", cmname)
	return fmt.Errorf("Timeout for checking configmap version in 10m")
}

func GetClientset() *kubernetes.Clientset {
	//config := &rest.Config{
	//}
	//config.Insecure = true
	//config.Host = "https://172.32.150.82:6443"
	//config.BearerToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJwcm9tZXRoZXVzLXRva2VuLTZrOWZyIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQubmFtZSI6InByb21ldGhldXMiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiI5ZjdjMjE5Ny0xN2QzLTExZTktYWU2MC01MjU0MDA4ZmRlMGYiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6a3ViZS1zeXN0ZW06cHJvbWV0aGV1cyJ9.YsHA4POxMzBST13hkLkHGmPebJ82ZzaFOigJXXis-HG2h3FnOAYM3ANoYV2Fzq3QNVJbVvXMwfjeVI5Q9WNu5Qca84viKlETHrMzjrDTkfelLTPGmaHEGFPoJGMwiwvKokwJAbjKdS_sGzNF6q8kIMFdGWWBIgzQmLwc4TF2yghuYe-cA5oyRnFloFJQa5awEsb3K5umZ4uYWUm0KrGsYpbcyxGJZc7Rzq3wJA6QJFXxHqhI5SAQd0eYynsbkCvoitNwUdE5JAaA00MTaB2v8nwR_qHhrXmsYVxB6IHg_CM8guvRBrc9mqMj7bPlhABjCKFfeNCtorCT2txiGd76QQ"

	config, _ := rest.InClusterConfig()
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	return clientset
}
