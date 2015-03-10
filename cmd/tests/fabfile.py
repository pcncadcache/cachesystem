from fabric.api import *
import yaml

def load_conf(filename):
    return yaml.load(open(filename).read())

def start(filename="config.yml"):
    local("../cache-server --port 6381 --daemonize yes")
    for name, instance in load_conf(filename).items():
        if "proxy" in name:
            local("../cache-proxy -d -c {0} --addr={1} --http-addr={2}".format(instance["config"], instance["address"], instance["http-address"]))
        elif "server" in name:
            local("../cache-server --maxmemory {0}mb --port {1} --daemonize yes".format(instance["maxmemory"], instance["port"]))
        else:
            print("Unknown instance name %s" % name)
def stop():
    
    with settings(warn_only=True):
        # local("kill $(pgrep cache-proxy)")
        local("kill $(pgrep cache-server)")

def status():
    # local("pgrep -l cache-proxy")
    local("pgrep -l cache-server")
