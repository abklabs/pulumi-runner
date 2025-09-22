import * as pulumi from "@pulumi/pulumi";
import * as inputs from "../types/input";
import * as outputs from "../types/output";
export declare class SSHDeployer extends pulumi.CustomResource {
    /**
     * Get an existing SSHDeployer resource's state with the given name, ID, and optional extra
     * properties used to qualify the lookup.
     *
     * @param name The _unique_ name of the resulting resource.
     * @param id The _unique_ provider ID of the resource to lookup.
     * @param opts Optional settings to control the behavior of the CustomResource.
     */
    static get(name: string, id: pulumi.Input<pulumi.ID>, opts?: pulumi.CustomResourceOptions): SSHDeployer;
    /**
     * Returns true if the given object is an instance of SSHDeployer.  This is designed to work even
     * when multiple copies of the Pulumi SDK have been loaded into the same process.
     */
    static isInstance(obj: any): obj is SSHDeployer;
    readonly connection: pulumi.Output<outputs.ssh.Connection>;
    readonly create: pulumi.Output<outputs.runner.CommandDefinition | undefined>;
    readonly delete: pulumi.Output<outputs.runner.CommandDefinition | undefined>;
    readonly environment: pulumi.Output<{
        [key: string]: string;
    } | undefined>;
    readonly payload: pulumi.Output<outputs.runner.FileAsset[] | undefined>;
    readonly update: pulumi.Output<outputs.runner.CommandDefinition | undefined>;
    /**
     * Create a SSHDeployer resource with the given unique name, arguments, and options.
     *
     * @param name The _unique_ name of the resource.
     * @param args The arguments to use to populate this resource's properties.
     * @param opts A bag of options that control this resource's behavior.
     */
    constructor(name: string, args: SSHDeployerArgs, opts?: pulumi.CustomResourceOptions);
}
/**
 * The set of arguments for constructing a SSHDeployer resource.
 */
export interface SSHDeployerArgs {
    connection: pulumi.Input<inputs.ssh.ConnectionArgs>;
    create?: pulumi.Input<inputs.runner.CommandDefinitionArgs>;
    delete?: pulumi.Input<inputs.runner.CommandDefinitionArgs>;
    environment?: pulumi.Input<{
        [key: string]: pulumi.Input<string>;
    }>;
    payload?: pulumi.Input<pulumi.Input<inputs.runner.FileAssetArgs>[]>;
    update?: pulumi.Input<inputs.runner.CommandDefinitionArgs>;
}
