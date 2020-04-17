package consts

const BaselineConfig = `
admin:
  access_log_path: /tmp/admin_access.log
  address:
    socket_address:
      address: 127.0.0.1
      port_value: {{.AdminPort}}
node:
  cluster: {{.NodeCluster}}
  id: {{.NodeId}}

dynamic_resources:
  lds_config:
    api_config_source:
      api_type: GRPC
      set_node_on_first_message_only: true
      grpc_services:
        - envoy_grpc:
            cluster_name: xds
static_resources:
  clusters:
    - connect_timeout: 1s
      hosts:
        - socket_address:
            address: {{.XdsAddress}}
            port_value: {{.XdsPort}}
      http2_protocol_options: {}
      name: xds
      type: STATIC
    - name: {{.CanaryClusterName}}
      connect_timeout: 1s
      type: EDS
      lb_policy: round_robin
      http2_protocol_options: {}
      eds_cluster_config:
        eds_config:
          api_config_source:
            api_type: GRPC
            set_node_on_first_message_only: true
            grpc_services:
              - envoy_grpc:
                  cluster_name: xds
    - name: {{.ServiceClusterName}}
      connect_timeout: 1s
      type: EDS
      lb_policy: round_robin
      http2_protocol_options: {}
      eds_cluster_config:
        eds_config:
          api_config_source:
            api_type: GRPC
            set_node_on_first_message_only: true
            grpc_services:
              - envoy_grpc:
                  cluster_name: xds
`
