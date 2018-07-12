#! /bin/sh

cat << EOF > /etc/consul.d/config.hcl
data_dir= "/tmp/consul"

ports {
  serf_lan = 9301
}

connect {
  enabled = true
  proxy {
   allow_managed_api_registration = true
   allow_managed_root = true
  }
}

retry_interval = "1s"
EOF

cat << EOF > /etc/consul.d/service.hcl
service {
  name = "http-echo"
  port = 8080


  connect {
    proxy {
      config {
        bind_port = 8443
      }
    }
  }
}
EOF

nohup sh -c "/http-echo -listen 127.0.0.1:8080 -text ='Hello World'" > /dev/null 2>&1 &
consul agent -config-dir /etc/consul.d -retry-join=consul.service.consul -advertise=192.168.192.131
