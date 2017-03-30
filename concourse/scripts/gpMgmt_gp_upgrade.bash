#!/bin/bash -l

set -eox pipefail

CWDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "${CWDIR}/common.bash"

function gen_env(){
  cat > /opt/run_test.sh <<-EOF
		base_path=\${1}
		source /usr/local/greenplum-db-devel/greenplum_path.sh
		source /opt/gcc_env.sh
		source \${base_path}/gpdb_src/gpAux/gpdemo/gpdemo-env.sh
		cd \${base_path}/gpdb_src/gpMgmt/go-utils/src/gp_upgrade
		make test
	EOF

	chmod a+x /opt/run_test.sh
}

function setup_gpadmin_user() {
    ./gpdb_src/concourse/scripts/setup_gpadmin_user.bash
}

function install_golang() {
# todo change to container or AMI
    su -c "./gpdb_src/concourse/scripts/install_golang.bash --64" gpadmin
}

function _main() {

    configure
    install_gpdb
    setup_gpadmin_user
    make_cluster
    gen_env
    install_golang
    run_test
}

_main "$@"