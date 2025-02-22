package init_watch

import (
	"github.com/goodrain/rainbond-task-plug/task-plug-producer/handle"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	barchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"strings"
	"sync"
)

type ManagerWatch interface {
	Start()
	OnAdd(obj interface{}, isInInitialList bool)
	OnDelete(obj interface{})
	GetApp(kind string) *App
	GetLister() *Lister
	OnUpdate(oldObj interface{}, newObj interface{})
	Ready() bool
}

type managerWatch struct {
	informers *Informer
	listers   *Lister
	stopch    chan struct{}
	app       sync.Map
}

func (mw *managerWatch) Start() {
	stopch := make(chan struct{})
	mw.informers.Start(stopch)
	mw.stopch = stopch
	for !mw.Ready() {
	}
}

func (mw *managerWatch) getApp(namespace string) *App {
	var app *App
	app = mw.GetApp(namespace)
	if app == nil {
		app = InitCacheApp()
		mw.app.Store(namespace, app)
	}
	return app
}

func (mw *managerWatch) GetLister() *Lister {
	return mw.listers
}

func (mw *managerWatch) GetApp(namespace string) *App {
	app, ok := mw.app.Load(namespace)
	if ok {
		app := app.(*App)
		return app
	}
	return nil
}

func (mw *managerWatch) OnAdd(obj interface{}, isInInitialList bool) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		app := mw.getApp(deployment.GetNamespace())
		app.SetDeployment(deployment)
		return
	}
	if statefulSet, ok := obj.(*appsv1.StatefulSet); ok {
		app := mw.getApp(statefulSet.GetNamespace())
		app.SetStatefulSet(statefulSet)
		return
	}
	if pod, ok := obj.(*v1.Pod); ok {
		app := mw.getApp(pod.GetNamespace())
		app.SetPod(pod)
		if pod.Annotations["normative_inspection"] == "open" {
			logrus.Infof("begin create normative inspection task")
			componentID := pod.Labels["service_id"]
			extendMethod := pod.Labels["extend_method"]
			err := handle.GetDispatchTasksHandle().CreateNormativeInspectionTask(componentID, extendMethod)
			if err != nil {
				logrus.Errorf("create source code inspection task failure: %v", err)
			}
		}
		if pod.Labels["job"] == "codebuild" && strings.HasSuffix(pod.Name, "dockerfile") {
			if pod.Annotations["code_inspection"] == "open" && pod.Annotations["repository_url"] != "" {
				logrus.Infof("begin create code inspection task")
				projectName := pod.Labels["service"]
				url := pod.Annotations["repository_url"]
				branch := pod.Annotations["repository_branch"]
				username := pod.Annotations["repository_username"]
				password := pod.Annotations["repository_password"]
				err := handle.GetDispatchTasksHandle().CreateSourceCodeInspectionTask(projectName, url, branch, username, password)
				if err != nil {
					logrus.Errorf("create source code inspection task failure: %v", err)
				}
			}
		}
		return
	}
	if svc, ok := obj.(*v1.Service); ok {
		app := mw.getApp(svc.GetNamespace())
		app.SetService(svc)
		return
	}
	if pvc, ok := obj.(*v1.PersistentVolumeClaim); ok {
		app := mw.getApp(pvc.GetNamespace())
		app.SetPVC(pvc)
		return
	}
	if cm, ok := obj.(*v1.ConfigMap); ok {
		app := mw.getApp(cm.GetNamespace())
		app.SetConfigMap(cm)
		return
	}
	if job, ok := obj.(*barchv1.Job); ok {
		app := mw.getApp(job.GetNamespace())
		app.SetJob(job)
	}
}

func (mw *managerWatch) OnDelete(obj interface{}) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		app := mw.getApp(deployment.GetNamespace())
		app.DeleteDeployment(deployment)
		return
	}
	if statefulSet, ok := obj.(*appsv1.StatefulSet); ok {
		app := mw.getApp(statefulSet.GetNamespace())
		app.DeleteStatefulSet(statefulSet)
		return
	}
	if pod, ok := obj.(*v1.Pod); ok {
		app := mw.getApp(pod.GetNamespace())
		app.DeletePod(pod)
		return
	}
	if pvc, ok := obj.(*v1.PersistentVolumeClaim); ok {
		app := mw.getApp(pvc.GetNamespace())
		app.DeletePVC(pvc)
		return
	}
	if cm, ok := obj.(*v1.ConfigMap); ok {
		app := mw.getApp(cm.GetNamespace())
		app.DeleteConfigMap(cm)
		return
	}
	if svc, ok := obj.(*v1.Service); ok {
		app := mw.getApp(svc.GetNamespace())
		app.DeleteService(svc)
		return
	}
}

func (mw *managerWatch) OnUpdate(oldObj interface{}, newObj interface{}) {
	//mw.OnAdd(newObj, false)
}

func (mw *managerWatch) Ready() bool {
	return mw.informers.Ready()
}
