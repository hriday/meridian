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
	export class IPReport {
	    input: string;
	    version: string;
	    coveringPrefix: string;
	    originAsn: string;
	    originName: string;
	    registeredOrg: string;
	    abuseEmail: string;
	    registry: string;
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

