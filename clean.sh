kubectl delete deploy -l domain=cloud.kage && kubectl delete cm -l domain=cloud.kage
kubectl delete -f test.yml --grace-period=1
kubectl apply -f test.yml
