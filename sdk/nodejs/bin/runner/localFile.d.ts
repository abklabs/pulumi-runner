import * as pulumi from "@pulumi/pulumi";
export declare function localFile(args?: LocalFileArgs, opts?: pulumi.InvokeOptions): Promise<LocalFileResult>;
export interface LocalFileArgs {
    contents?: string;
    filename?: string;
    localPath?: string;
    mode?: number;
}
export interface LocalFileResult {
    readonly contents?: string;
    readonly filename?: string;
    readonly localPath?: string;
    readonly mode?: number;
}
export declare function localFileOutput(args?: LocalFileOutputArgs, opts?: pulumi.InvokeOutputOptions): pulumi.Output<LocalFileResult>;
export interface LocalFileOutputArgs {
    contents?: pulumi.Input<string>;
    filename?: pulumi.Input<string>;
    localPath?: pulumi.Input<string>;
    mode?: pulumi.Input<number>;
}
