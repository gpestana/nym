//go:generate go run $GOPATH/src/github.com/nymtech/nym/gen/queueGenMain.go --name=JobQueue --type=*jobpacket.JobPacket --typeName=JobPacket --typeImportPath=github.com/nymtech/nym/crypto/coconut/concurrency/jobpacket

// jobqueue.go - Entry point for go generate to create a job queue.
// Copyright (C) 2019  Jedrzej Stuczynski.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package jobqueue
