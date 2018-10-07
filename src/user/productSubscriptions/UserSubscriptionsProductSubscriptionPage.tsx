import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { gql, queryGraphQL } from '@sourcegraph/webapp/dist/backend/graphql'
import * as GQL from '@sourcegraph/webapp/dist/backend/graphqlschema'
import { PageTitle } from '@sourcegraph/webapp/dist/components/PageTitle'
import { SiteAdminAlert } from '@sourcegraph/webapp/dist/site-admin/SiteAdminAlert'
import { eventLogger } from '@sourcegraph/webapp/dist/tracking/eventLogger'
import { asError, createAggregateError, ErrorLike, isErrorLike } from '@sourcegraph/webapp/dist/util/errors'
import * as React from 'react'
import { RouteComponentProps } from 'react-router'
import { Link } from 'react-router-dom'
import { Observable, Subject, Subscription } from 'rxjs'
import { catchError, distinctUntilChanged, map, startWith, switchMap } from 'rxjs/operators'
import { mailtoSales } from '../../productSubscription/helpers'
import { BackToAllSubscriptionsLink } from './BackToAllSubscriptionsLink'
import { ProductSubscriptionBilling } from './ProductSubscriptionBilling'
import { ProductSubscriptionHistory } from './ProductSubscriptionHistory'
import { UserProductSubscriptionStatus } from './UserProductSubscriptionStatus'

interface Props extends RouteComponentProps<{ subscriptionID: string }> {
    user: GQL.IUser
}

const LOADING: 'loading' = 'loading'

interface State {
    /**
     * The product subscription, or loading, or an error.
     */
    productSubscriptionOrError: typeof LOADING | GQL.IProductSubscription | ErrorLike
}

/**
 * Displays a product subscription in the user subscriptions area.
 */
export class UserSubscriptionsProductSubscriptionPage extends React.Component<Props, State> {
    public state: State = { productSubscriptionOrError: LOADING }

    private componentUpdates = new Subject<Props>()
    private subscriptions = new Subscription()

    public componentDidMount(): void {
        eventLogger.logViewEvent('UserSubscriptionsProductSubscription')

        const subscriptionIDChanges = this.componentUpdates.pipe(
            map(props => props.match.params.subscriptionID),
            distinctUntilChanged()
        )

        const productSubscriptionChanges = subscriptionIDChanges.pipe(
            switchMap(subscriptionID =>
                this.queryProductSubscription(subscriptionID).pipe(
                    catchError(err => [asError(err)]),
                    startWith(LOADING)
                )
            )
        )

        this.subscriptions.add(
            productSubscriptionChanges
                .pipe(map(result => ({ productSubscriptionOrError: result })))
                .subscribe(stateUpdate => this.setState(stateUpdate))
        )

        this.componentUpdates.next(this.props)
    }

    public componentDidUpdate(): void {
        this.componentUpdates.next(this.props)
    }

    public componentWillUnmount(): void {
        this.subscriptions.unsubscribe()
    }

    public render(): JSX.Element | null {
        return (
            <div className="user-subscriptions-product-subscription-page">
                <PageTitle title="Subscription" />
                <div className="d-flex align-items-center justify-content-between">
                    <BackToAllSubscriptionsLink user={this.props.user} />
                    {this.state.productSubscriptionOrError !== LOADING &&
                        !isErrorLike(this.state.productSubscriptionOrError) &&
                        this.state.productSubscriptionOrError.urlForSiteAdmin && (
                            <SiteAdminAlert className="small m-0 p-1">
                                <Link
                                    to={this.state.productSubscriptionOrError.urlForSiteAdmin}
                                    className="mt-2 d-block"
                                >
                                    View subscription
                                </Link>
                            </SiteAdminAlert>
                        )}
                </div>
                {this.state.productSubscriptionOrError === LOADING ? (
                    <LoadingSpinner className="icon-inline" />
                ) : isErrorLike(this.state.productSubscriptionOrError) ? (
                    <div className="alert alert-danger my-2">
                        Error: {this.state.productSubscriptionOrError.message}
                    </div>
                ) : (
                    <div className="row">
                        <div className="col-md-9">
                            <h2>Subscription {this.state.productSubscriptionOrError.name}</h2>
                            {(this.state.productSubscriptionOrError.plan ||
                                (this.state.productSubscriptionOrError.activeLicense &&
                                    this.state.productSubscriptionOrError.activeLicense.info)) && (
                                <UserProductSubscriptionStatus
                                    subscriptionName={this.state.productSubscriptionOrError.name}
                                    productNameWithBrand={
                                        this.state.productSubscriptionOrError.plan
                                            ? this.state.productSubscriptionOrError.plan.nameWithBrand
                                            : this.state.productSubscriptionOrError.activeLicense!.info!
                                                  .productNameWithBrand
                                    }
                                    userCount={
                                        this.state.productSubscriptionOrError.userCount !== null
                                            ? this.state.productSubscriptionOrError.userCount
                                            : this.state.productSubscriptionOrError.activeLicense!.info!.userCount
                                    }
                                    expiresAt={
                                        this.state.productSubscriptionOrError.expiresAt !== null
                                            ? this.state.productSubscriptionOrError.expiresAt
                                            : this.state.productSubscriptionOrError.activeLicense!.info!.expiresAt
                                    }
                                    licenseKey={
                                        this.state.productSubscriptionOrError.activeLicense &&
                                        this.state.productSubscriptionOrError.activeLicense.licenseKey
                                    }
                                />
                            )}
                            <div className="card mt-3">
                                <div className="card-header">Billing</div>
                                {this.state.productSubscriptionOrError.invoiceItem ? (
                                    <>
                                        <ProductSubscriptionBilling
                                            productSubscription={this.state.productSubscriptionOrError}
                                        />
                                        <div className="card-footer">
                                            <a
                                                href={mailtoSales({
                                                    subject: `No license key for subscription ${
                                                        this.state.productSubscriptionOrError.name
                                                    }`,
                                                })}
                                            >
                                                Contact sales
                                            </a>{' '}
                                            to change your payment method.
                                        </div>
                                    </>
                                ) : (
                                    <div className="card-body">
                                        <span className="text-muted ">
                                            No billing information is associated with this subscription.{' '}
                                            <a
                                                href={mailtoSales({
                                                    subject: `Billing for subscription ${
                                                        this.state.productSubscriptionOrError.name
                                                    }`,
                                                })}
                                            >
                                                Contact sales
                                            </a>{' '}
                                            for help.
                                        </span>
                                    </div>
                                )}
                            </div>
                            <div className="card mt-3">
                                <div className="card-header">History</div>
                                <ProductSubscriptionHistory
                                    productSubscription={this.state.productSubscriptionOrError}
                                />
                            </div>
                        </div>
                    </div>
                )}
            </div>
        )
    }

    private queryProductSubscription = (id: GQL.ID): Observable<GQL.IProductSubscription> =>
        queryGraphQL(
            gql`
                query ProductSubscription($id: ID!) {
                    node(id: $id) {
                        ... on ProductSubscription {
                            ...ProductSubscriptionFields
                        }
                    }
                }
                fragment ProductSubscriptionFields on ProductSubscription {
                    id
                    name
                    account {
                        id
                        username
                        displayName
                        emails {
                            email
                            verified
                        }
                    }
                    plan {
                        billingPlanID
                        name
                        nameWithBrand
                        pricePerUserPerYear
                    }
                    userCount
                    expiresAt
                    events {
                        id
                        date
                        title
                        description
                        url
                    }
                    activeLicense {
                        licenseKey
                        info {
                            productNameWithBrand
                            tags
                            userCount
                            expiresAt
                        }
                    }
                    createdAt
                    isArchived
                    url
                    urlForSiteAdmin
                }
            `,
            { id }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.node || (errors && errors.length > 0)) {
                    throw createAggregateError(errors)
                }
                return data.node as GQL.IProductSubscription
            })
        )
}