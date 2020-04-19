#!/bin/bash

cat << EOF 
---
apiVersion: v1
kind: Secret
metadata:
  name: go-ecobee-config
type: Opaque
stringData:
  config.yaml: |-
$(cat config.yaml | awk '{print "    "$0}')
---
apiVersion: v1
kind: Secret
metadata:
  name: go-ecobee-varauth
type: Opaque
stringData:
  authcache.json: |-
$(cat authcache | awk '{print "    "$0}')
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-ecobee
  labels:
    run: go-ecobee-promsvr
spec:
  replicas: 1
  selector:
    matchLabels:
      run: go-ecobee-promsvr
  template:
    metadata:
      labels:
        run: go-ecobee-promsvr
    spec:
      initContainers:
      - name: set-key-ownership
        image: alpine:3.6
        command: ["sh", "-c", "cp /app/go-ecobee/tmpvar/* /app/go-ecobee/var && chown -R 100 /app/go-ecobee/var"]
        volumeMounts:
        - mountPath: "/app/go-ecobee/var"
          name: go-ecobee-var
        - mountPath: "/app/go-ecobee/tmpvar"
          name: go-ecobee-varauth-temp
      containers:
      - name: prom-server
        image: rraboy/go-ecobee
        imagePullPolicy: Always
        args:
        - "promsvr"
        - "--config"
        - "/app/go-ecobee/etc/config.yaml"
        - "--authcache"
        - "/app/go-ecobee/var/authcache.json"
        volumeMounts:
        - name: go-ecobee-etc
          mountPath: "/app/go-ecobee/etc"
          readOnly: false
        - name: go-ecobee-var
          mountPath: "/app/go-ecobee/var"
          readOnly: false
        ports:
        - containerPort: 9442
      volumes:
      - name: go-ecobee-etc
        secret:
          secretName: go-ecobee-config
      - name: go-ecobee-varauth-temp
        secret:
          secretName: go-ecobee-varauth
      - name: go-ecobee-var
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: go-ecobee-promsvr
spec:
  type: ClusterIP
  ports:
  - port: 9442
    protocol: TCP
    targetPort: 9442
  selector:
    run: go-ecobee-promsvr
EOF
