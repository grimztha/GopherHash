export namespace main {
	
	export class FileCompareResult {
	    File1: string;
	    File2: string;
	    File1Name: string;
	    File2Name: string;
	    File1Size: number;
	    File2Size: number;
	    Match: boolean;
	    Algorithm: string;
	    Hash1: string;
	    Hash2: string;
	    Duration: number;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new FileCompareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.File1 = source["File1"];
	        this.File2 = source["File2"];
	        this.File1Name = source["File1Name"];
	        this.File2Name = source["File2Name"];
	        this.File1Size = source["File1Size"];
	        this.File2Size = source["File2Size"];
	        this.Match = source["Match"];
	        this.Algorithm = source["Algorithm"];
	        this.Hash1 = source["Hash1"];
	        this.Hash2 = source["Hash2"];
	        this.Duration = source["Duration"];
	        this.Error = source["Error"];
	    }
	}
	export class HashResult {
	    FileName: string;
	    Hash: string;
	    Algorithm: string;
	    Size: number;
	    Duration: number;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new HashResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FileName = source["FileName"];
	        this.Hash = source["Hash"];
	        this.Algorithm = source["Algorithm"];
	        this.Size = source["Size"];
	        this.Duration = source["Duration"];
	        this.Error = source["Error"];
	    }
	}
	export class VerificationResult {
	    FileName: string;
	    Expected: string;
	    Actual: string;
	    Algorithm: string;
	    Valid: boolean;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new VerificationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FileName = source["FileName"];
	        this.Expected = source["Expected"];
	        this.Actual = source["Actual"];
	        this.Algorithm = source["Algorithm"];
	        this.Valid = source["Valid"];
	        this.Error = source["Error"];
	    }
	}

}

