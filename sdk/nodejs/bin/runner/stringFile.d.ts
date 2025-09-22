import * as pulumi from "@pulumi/pulumi";
export declare function stringFile(args?: StringFileArgs, opts?: pulumi.InvokeOptions): Promise<StringFileResult>;
export interface StringFileArgs {
    contents?: string;
    filename?: string;
    localPath?: string;
    mode?: number;
}
export interface StringFileResult {
    readonly contents?: string;
    readonly filename?: string;
    readonly localPath?: string;
    readonly mode?: number;
}
export declare function stringFileOutput(args?: StringFileOutputArgs, opts?: pulumi.InvokeOutputOptions): pulumi.Output<StringFileResult>;
export interface StringFileOutputArgs {
    contents?: pulumi.Input<string>;
    filename?: pulumi.Input<string>;
    localPath?: pulumi.Input<string>;
    mode?: pulumi.Input<number>;
}
