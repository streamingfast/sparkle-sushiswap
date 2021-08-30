#!/bin/bash
set -x;

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"
STOREURL=gs://dfuseio-global-blocks-us/eth-mainnet/v5
RPCENDPOINT="https://bitter-withered-forest.quiknode.pro/750d6d73a803c919af639034b82ef675975cb2ba/"
if test -d ../localblocks; then
  echo "Using blocks from local store: ./localblocks"
    STOREURL=./localblocks
  else
    echo "Fetching blocks from remote store. You should copy them locally to make this faster..., ex:"
    cat <<EOC
######

mkdir ./localblocks
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001079*" ./localblocks/
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001080*" ./localblocks/
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001081*" ./localblocks/
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001082*" ./localblocks/
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001083*" ./localblocks/
gsutil -m cp "gs://dfuseio-global-blocks-us/eth-mainnet/v5/001084*" ./localblocks/

######
EOC
fi

function step1() {
    INFO=.* exchange parallel step -s 1 --output-path ./step1-v1 --start-block $1 --stop-block $2 --blocks-store-url $STOREURL --rpc-endpoint $RPCENDPOINT &
}
function step2() {
    INFO=.* exchange parallel step -s 2 --input-path ./step1-v1 --output-path ./step2-v1 --start-block $1 --stop-block $2 --blocks-store-url $STOREURL --rpc-endpoint $RPCENDPOINT &
}
function step3() {
    INFO=.* exchange parallel step -s 3 --input-path ./step2-v1 --output-path ./step3-v1 --start-block $1 --stop-block $2 --blocks-store-url $STOREURL --rpc-endpoint $RPCENDPOINT &
}
function step4() {
    INFO=.* exchange parallel step -s 4 --flush-entities --store-snapshot=false --input-path ./step3-v1 --output-path ./step4-v1  --start-block $1 --stop-block $2  --blocks-store-url $STOREURL --rpc-endpoint $RPCENDPOINT &
}


main() {
  pushd "$ROOT" &> /dev/null
    go install -v ./cmd/exchange || exit 1

    if [ "$1" != "" ] && [ "$1" != 1 ]; then
      echo "SKIPPING STEP 1"
    else
      echo "LAUNCHING STEP 1"
      rm -rf ./step1-v1


      step1 10794229 10804228
      step1 10804229 10814228
      step1 10814229 10824228
      step1 10824229 10829546

      for job in `jobs -p`; do
          echo "Waiting on $job"
          wait $job
      done
    fi

    if [ "$1" != "" ] && [ "$1" != 2 ]; then
      echo "SKIPPING STEP 2"
    else
      echo "LAUNCHING STEP 2"
      rm -rf ./step2-v1

      step2 10794229 10804228
      step2 10804229 10814228
      step2 10814229 10824228
      step2 10824229 10829546

      for job in `jobs -p`; do
          echo "Waiting on $job"
          wait $job
      done
    fi

    if [ "$1" != "" ] && [ "$1" != 3 ]; then
      echo "SKIPPING STEP 3"
    else
      echo "LAUNCHING STEP 3"
      rm -rf ./step3-v1

      step3 10794229 10804228
      step3 10804229 10814228
      step3 10814229 10824228
      step3 10824229 10829546

      for job in `jobs -p`; do
          echo "Waiting on $job"
          wait $job
      done
    fi

    if [ "$1" != "" ] && [ "$1" != 4 ]; then
      echo "SKIPPING STEP 4"
    else
      echo "LAUNCHING STEP 4"
      rm -rf ./step4-v1

      step4 10794229 10804228
      step4 10804229 10814228
      step4 10814229 10824228
      step4 10824229 10829546

      for job in `jobs -p`; do
          echo "Waiting on $job"
          wait $job
      done
    fi

    if [ "$1" != "" ] && [ "$1" != csv ]; then
      echo "SKIPPING STEP CSV"
    else
      echo "Exporting to csv"
      INFO=.* exchange parallel to-csv --input-path ./step4-v1 --output-path ./stepcsvs --chunk-size 1000
    fi
  popd &> /dev/null
}

main $@
