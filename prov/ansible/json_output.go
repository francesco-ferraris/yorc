package ansible

import (
	"bytes"
	"fmt"
	"path"
	"time"

	"strconv"

	"github.com/antonholmquist/jason"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/deployments"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

func getAnsibleJsonResult(output *bytes.Buffer) (*jason.Object, error) {
	// Workaround https://github.com/ansible/ansible/issues/17122
	b := output.Bytes()
	if i := bytes.Index(b, []byte("{")); i >= 0 {
		b = b[i:]
	} else {
		return nil, errors.New("Not a valid JSON output")
	}
	//Construct the JSON from the buffer
	v, err := jason.NewObjectFromBytes(b)
	if err != nil {
		err = errors.Wrap(err, "Ansible logs not available")
		log.Printf("%v", err)
		log.Debugf("%+v", err)
		log.Debugf("String: %q", string(b))
		return nil, err
	}
	return v, nil
}

func (e *executionCommon) logAnsibleOutputInConsul(output *bytes.Buffer) error {

	v, err := getAnsibleJsonResult(output)
	if err != nil {
		return err
	}

	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	for _, play := range plays {
		//Extract the tasks from the play
		tasks, err := play.GetObjectArray("tasks")
		if err != nil {
			err = errors.Wrap(err, "Ansible logs not available")
			log.Printf("%v", err)
			log.Debugf("%+v", err)
			continue
		}
		for _, task := range tasks {
			//Extract the hosts object from the  tasks
			tmp, err := task.GetObject("hosts")
			if err != nil {
				err = errors.Wrap(err, "Ansible logs not available")
				log.Printf("%v", err)
				log.Debugf("%+v", err)
				continue
			}
			//Convert the host into map like ["IP_ADDR"]Json_Object
			mapTmp := tmp.Map()
			//Iterate on this map (normally a single object)
			for host, v := range mapTmp {
				//Convert the value in Object type
				obj, err := v.Object()
				if err != nil {
					err = errors.Wrap(err, "Ansible logs not available")
					log.Printf("%v", err)
					log.Debugf("%+v", err)
					continue
				}
				//Check if a stderr field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stderr"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stderr found on host : %s  message : %s", host, std)
					key := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, stderr:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
				//Check if a stdout field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("stdout"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stdout found on host : %s  message : %s", host, std)
					key := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, stdout:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}

				//Check if a msg field is present (The stdout field is exported for shell tasks on ansible)
				if std, err := obj.GetString("msg"); err == nil && std != "" {
					//Display it and store it in consul
					log.Debugf("Stdout found on host : %s  message : %s", host, std)
					key := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
					err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, host %q, msg:\n%s", e.NodeName, host, std))
					if err != nil {
						err = errors.Wrap(err, "Ansible logs not available")
						log.Printf("%v", err)
						log.Debugf("%+v", err)
						continue
					}
				}
			}
		}

	}

	return nil
}

func (e *executionAnsible) logAnsibleOutputInConsul(output *bytes.Buffer) error {

	v, err := getAnsibleJsonResult(output)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	//Get the array of object of plays
	plays, err := v.GetObjectArray("plays")
	for _, play := range plays {
		playName, err := play.GetString("play", "name")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play name")
		}
		buf.WriteString("\nPlay [")
		buf.WriteString(playName)
		buf.WriteString("]")
		log.Debugf("Play name is %q", playName)

		//Extract the tasks from the play
		tasks, err := play.GetObjectArray("tasks")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play tasks")
		}

		for _, task := range tasks {
			taskName, err := task.GetString("task", "name")
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve play tasks")
			}
			buf.WriteString("\n\nTask [")
			buf.WriteString(taskName)
			buf.WriteString("]")

			//Extract the hosts object from the  tasks
			hosts, err := task.GetObject("hosts")
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve hosts results for play task")
			}

			//Iterate on this map (normally a single object)
			for hostName, hostVal := range hosts.Map() {
				buf.WriteString("\n")
				//Convert the value in Object type
				host, err := hostVal.Object()
				if err != nil {
					return errors.Wrapf(err, "Failed to retrieve task result for host %q", hostName)
				}
				if failed, err := host.GetBoolean("failed"); err == nil && failed {
					buf.WriteString("failed: [")
				} else if unreachable, err := host.GetBoolean("unreachable"); err == nil && unreachable {
					buf.WriteString("unreachable: [")
				} else if skipped, err := host.GetBoolean("skipped"); err == nil && skipped {
					buf.WriteString("skipped: [")
				} else if changed, err := host.GetBoolean("changed"); err == nil && changed {
					buf.WriteString("changed: [")
				} else {
					buf.WriteString("ok: [")
				}
				buf.WriteString(hostName)
				buf.WriteString("]")
				if msg, err := host.GetString("msg"); err == nil && msg != "" {
					buf.WriteString(" => {\n\tmsg: \"")
					buf.WriteString(msg)
					buf.WriteString("\"\n}")

				}
			}

		}
	}

	buf.WriteString("\nStats:")
	stats, err := v.GetObject("stats")
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve play stats")
	}
	for statsHost, statsValue := range stats.Map() {
		buf.WriteString("\nHost: ")
		buf.WriteString(statsHost)
		statsObj, err := statsValue.Object()
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		changed, err := statsObj.GetInt64("changed")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" changed: ")
		buf.WriteString(strconv.FormatInt(changed, 10))
		failures, err := statsObj.GetInt64("failures")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" failures: ")
		buf.WriteString(strconv.FormatInt(failures, 10))
		ok, err := statsObj.GetInt64("ok")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" ok: ")
		buf.WriteString(strconv.FormatInt(ok, 10))
		skipped, err := statsObj.GetInt64("skipped")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" skipped: ")
		buf.WriteString(strconv.FormatInt(skipped, 10))
		unreachable, err := statsObj.GetInt64("unreachable")
		if err != nil {
			return errors.Wrap(err, "Failed to retrieve play stats")
		}
		buf.WriteString(" unreachable: ")
		buf.WriteString(strconv.FormatInt(unreachable, 10))

	}
	key := path.Join(deployments.DeploymentKVPrefix, e.DeploymentId, "logs", deployments.SOFTWARE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano))
	err = consulutil.StoreConsulKeyAsString(key, fmt.Sprintf("node %q, Ansible Playbook result:\n%s", e.NodeName, buf.String()))
	if err != nil {
		return err
	}

	return nil
}