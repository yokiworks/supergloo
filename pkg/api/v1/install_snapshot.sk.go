// Code generated by protoc-gen-solo-kit. DO NOT EDIT.

package v1

import (
	encryption_istio_io "github.com/solo-io/supergloo/pkg/api/external/istio/encryption/v1"

	"github.com/mitchellh/hashstructure"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"go.uber.org/zap"
)

type InstallSnapshot struct {
	Installs   InstallsByNamespace
	Istiocerts encryption_istio_io.IstiocertsByNamespace
}

func (s InstallSnapshot) Clone() InstallSnapshot {
	return InstallSnapshot{
		Installs:   s.Installs.Clone(),
		Istiocerts: s.Istiocerts.Clone(),
	}
}

func (s InstallSnapshot) snapshotToHash() InstallSnapshot {
	snapshotForHashing := s.Clone()
	for _, install := range snapshotForHashing.Installs.List() {
		resources.UpdateMetadata(install, func(meta *core.Metadata) {
			meta.ResourceVersion = ""
		})
		install.SetStatus(core.Status{})
	}
	for _, istioCacertsSecret := range snapshotForHashing.Istiocerts.List() {
		resources.UpdateMetadata(istioCacertsSecret, func(meta *core.Metadata) {
			meta.ResourceVersion = ""
		})
	}

	return snapshotForHashing
}

func (s InstallSnapshot) Hash() uint64 {
	return s.hashStruct(s.snapshotToHash())
}

func (s InstallSnapshot) HashFields() []zap.Field {
	snapshotForHashing := s.snapshotToHash()
	var fields []zap.Field
	installs := s.hashStruct(snapshotForHashing.Installs.List())
	fields = append(fields, zap.Uint64("installs", installs))
	istiocerts := s.hashStruct(snapshotForHashing.Istiocerts.List())
	fields = append(fields, zap.Uint64("istiocerts", istiocerts))

	return append(fields, zap.Uint64("snapshotHash", s.hashStruct(snapshotForHashing)))
}

func (s InstallSnapshot) hashStruct(v interface{}) uint64 {
	h, err := hashstructure.Hash(v, nil)
	if err != nil {
		panic(err)
	}
	return h
}
