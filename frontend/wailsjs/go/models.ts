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

