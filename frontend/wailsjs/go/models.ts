export namespace app {
	
	export class IPSpaceState {
	    exists: boolean;
	    unlocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IPSpaceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.unlocked = source["unlocked"];
	    }
	}

}

export namespace dnsx {
	
	export class Record {
	    name: string;
	    type: string;
	    ttl: number;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.ttl = source["ttl"];
	        this.value = source["value"];
	    }
	}
	export class Result {
	    server: string;
	    rcode: string;
	    flags: string[];
	    authenticated: boolean;
	    rttMillis: number;
	    answers: Record[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.rcode = source["rcode"];
	        this.flags = source["flags"];
	        this.authenticated = source["authenticated"];
	        this.rttMillis = source["rttMillis"];
	        this.answers = this.convertValues(source["answers"], Record);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace ipspace {
	
	export class Record {
	    id: string;
	    prefix: string;
	    name: string;
	    status: string;
	    tags: string[];
	    notes: string;
	    parentId: string;
	    createdBy: string;
	    changedBy: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    changedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.prefix = source["prefix"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.tags = source["tags"];
	        this.notes = source["notes"];
	        this.parentId = source["parentId"];
	        this.createdBy = source["createdBy"];
	        this.changedBy = source["changedBy"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.changedAt = this.convertValues(source["changedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Event {
	    id: string;
	    type: string;
	    actor: string;
	    // Go type: time
	    at: any;
	    record: Record;
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.actor = source["actor"];
	        this.at = this.convertValues(source["at"], null);
	        this.record = this.convertValues(source["record"], Record);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportReport {
	    added: number;
	    skipped: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	
	export class TreeNode {
	    record: Record;
	    children: TreeNode[];
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = this.convertValues(source["record"], Record);
	        this.children = this.convertValues(source["children"], TreeNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace lookup {
	
	export class Source {
	    label: string;
	    // Go type: time
	    fetchedAt: any;
	    err: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.fetchedAt = this.convertValues(source["fetchedAt"], null);
	        this.err = source["err"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ASNReport {
	    asn: string;
	    holder: string;
	    registry: string;
	    v4Prefixes: number;
	    v6Prefixes: number;
	    neighbours: number;
	    routingSrc: Source;
	    registrySrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new ASNReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asn = source["asn"];
	        this.holder = source["holder"];
	        this.registry = source["registry"];
	        this.v4Prefixes = source["v4Prefixes"];
	        this.v6Prefixes = source["v6Prefixes"];
	        this.neighbours = source["neighbours"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BGPCollector {
	    rrc: string;
	    location: string;
	    scope: string;
	    peers: number;
	
	    static createFrom(source: any = {}) {
	        return new BGPCollector(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rrc = source["rrc"];
	        this.location = source["location"];
	        this.scope = source["scope"];
	        this.peers = source["peers"];
	    }
	}
	export class BGPPath {
	    path: string[];
	    peers: number;
	    percent: number;
	
	    static createFrom(source: any = {}) {
	        return new BGPPath(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.peers = source["peers"];
	        this.percent = source["percent"];
	    }
	}
	export class BGPReport {
	    input: string;
	    prefix: string;
	    originAsns: string[];
	    moas: boolean;
	    collectors: number;
	    peers: number;
	    paths: BGPPath[];
	    upstreams: string[];
	    transit: string[];
	    rpkiState: string;
	    vantages: BGPCollector[];
	    routingSrc: Source;
	    rpkiSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new BGPReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.prefix = source["prefix"];
	        this.originAsns = source["originAsns"];
	        this.moas = source["moas"];
	        this.collectors = source["collectors"];
	        this.peers = source["peers"];
	        this.paths = this.convertValues(source["paths"], BGPPath);
	        this.upstreams = source["upstreams"];
	        this.transit = source["transit"];
	        this.rpkiState = source["rpkiState"];
	        this.vantages = this.convertValues(source["vantages"], BGPCollector);
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.rpkiSrc = this.convertValues(source["rpkiSrc"], Source);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DomainReport {
	    input: string;
	    rcode: string;
	    records: dnsx.Record[];
	    resolvesTo: string[];
	    hostedBy: string[];
	    registrar: string;
	    created: string;
	    expires: string;
	    dnssec: string;
	    nameservers: string[];
	    dnsSrc: Source;
	    registrySrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new DomainReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.rcode = source["rcode"];
	        this.records = this.convertValues(source["records"], dnsx.Record);
	        this.resolvesTo = source["resolvesTo"];
	        this.hostedBy = source["hostedBy"];
	        this.registrar = source["registrar"];
	        this.created = source["created"];
	        this.expires = source["expires"];
	        this.dnssec = source["dnssec"];
	        this.nameservers = source["nameservers"];
	        this.dnsSrc = this.convertValues(source["dnsSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IPReport {
	    input: string;
	    version: string;
	    coveringPrefix: string;
	    originAsn: string;
	    originName: string;
	    registeredOrg: string;
	    abuseEmail: string;
	    registry: string;
	    rpkiState: string;
	    routingNote: string;
	    routingSrc: Source;
	    registrySrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new IPReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.version = source["version"];
	        this.coveringPrefix = source["coveringPrefix"];
	        this.originAsn = source["originAsn"];
	        this.originName = source["originName"];
	        this.registeredOrg = source["registeredOrg"];
	        this.abuseEmail = source["abuseEmail"];
	        this.registry = source["registry"];
	        this.rpkiState = source["rpkiState"];
	        this.routingNote = source["routingNote"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PrefixReport {
	    input: string;
	    originAsn: string;
	    originName: string;
	    announced: boolean;
	    registry: string;
	    registeredOrg: string;
	    abuseEmail: string;
	    rpkiState: string;
	    rpkiRoa: string;
	    lessSpecific: string[];
	    moreSpecific: string[];
	    routingSrc: Source;
	    registrySrc: Source;
	    rpkiSrc: Source;
	
	    static createFrom(source: any = {}) {
	        return new PrefixReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.originAsn = source["originAsn"];
	        this.originName = source["originName"];
	        this.announced = source["announced"];
	        this.registry = source["registry"];
	        this.registeredOrg = source["registeredOrg"];
	        this.abuseEmail = source["abuseEmail"];
	        this.rpkiState = source["rpkiState"];
	        this.rpkiRoa = source["rpkiRoa"];
	        this.lessSpecific = source["lessSpecific"];
	        this.moreSpecific = source["moreSpecific"];
	        this.routingSrc = this.convertValues(source["routingSrc"], Source);
	        this.registrySrc = this.convertValues(source["registrySrc"], Source);
	        this.rpkiSrc = this.convertValues(source["rpkiSrc"], Source);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace netinput {
	
	export class Result {
	    Kind: number;
	    Normalized: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.Normalized = source["Normalized"];
	    }
	}

}

export namespace ntp {
	
	export class Reply {
	    server: string;
	    stratum: number;
	    refId: string;
	    leapIndicator: string;
	    precisionSeconds: number;
	    rootDelay: number;
	    rootDispersion: number;
	    // Go type: time
	    serverTime: any;
	    offset: number;
	    rtt: number;
	    kissOfDeath: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Reply(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.stratum = source["stratum"];
	        this.refId = source["refId"];
	        this.leapIndicator = source["leapIndicator"];
	        this.precisionSeconds = source["precisionSeconds"];
	        this.rootDelay = source["rootDelay"];
	        this.rootDispersion = source["rootDispersion"];
	        this.serverTime = this.convertValues(source["serverTime"], null);
	        this.offset = source["offset"];
	        this.rtt = source["rtt"];
	        this.kissOfDeath = source["kissOfDeath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace subnet {
	
	export class Info {
	    Prefix: string;
	    Version: number;
	    Network: string;
	    Broadcast: string;
	    Netmask: string;
	    Wildcard: string;
	    PrefixLen: number;
	    TotalCount: string;
	    UsableFirst: string;
	    UsableLast: string;
	    UsableCount: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Prefix = source["Prefix"];
	        this.Version = source["Version"];
	        this.Network = source["Network"];
	        this.Broadcast = source["Broadcast"];
	        this.Netmask = source["Netmask"];
	        this.Wildcard = source["Wildcard"];
	        this.PrefixLen = source["PrefixLen"];
	        this.TotalCount = source["TotalCount"];
	        this.UsableFirst = source["UsableFirst"];
	        this.UsableLast = source["UsableLast"];
	        this.UsableCount = source["UsableCount"];
	    }
	}

}

export namespace sysconf {
	
	export class Found {
	    values: string[];
	    source: string;
	    err: string;
	
	    static createFrom(source: any = {}) {
	        return new Found(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.values = source["values"];
	        this.source = source["source"];
	        this.err = source["err"];
	    }
	}

}

export namespace trace {
	
	export class Hop {
	    ttl: number;
	    addr: string;
	    host: string;
	    rtts: number[];
	    asn: string;
	    asOwner: string;
	    timeout: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Hop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ttl = source["ttl"];
	        this.addr = source["addr"];
	        this.host = source["host"];
	        this.rtts = source["rtts"];
	        this.asn = source["asn"];
	        this.asOwner = source["asOwner"];
	        this.timeout = source["timeout"];
	    }
	}
	export class Result {
	    target: string;
	    addr: string;
	    hops: Hop[];
	    reached: boolean;
	    elapsed: number;
	    verdict: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = source["target"];
	        this.addr = source["addr"];
	        this.hops = this.convertValues(source["hops"], Hop);
	        this.reached = source["reached"];
	        this.elapsed = source["elapsed"];
	        this.verdict = source["verdict"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

