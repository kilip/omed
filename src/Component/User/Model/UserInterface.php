<?php

/*
 * This file is part of the Omed project.
 *
 * (c) Anthonius Munthi <https://itstoni.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

declare(strict_types=1);

namespace Omed\Component\User\Model;

/**
 * User Interface.
 */
interface UserInterface
{
    public const ROLE_DEFAULT = 'ROLE_USER';

    public const ROLE_SUPER_ADMIN = 'ROLE_SUPER_ADMIN';

    /**
     * Check if user has role.
     *
     * @param string $role
     *
     * @return bool
     */
    public function hasRole(string $role);

    /**
     * Sets the roles of the user.
     *
     * This overwrites any previous roles.
     *
     * @param array $roles
     *
     * @return static
     */
    public function setRoles(array $roles);

    /**
     * Adds a role to the user.
     *
     * @param string $role
     *
     * @return static
     */
    public function addRole(string $role);

    /**
     * Removes a role to the user.
     *
     * @param string $role
     *
     * @return static
     */
    public function removeRole(string $role);

    /**
     * Check if user is enabled.
     *
     * @return bool
     */
    public function isEnabled(): bool;

    /**
     * Set user enabled.
     *
     * @param bool $enabled
     *
     * @return static
     */
    public function setEnabled(bool $enabled);

    /**
     * Removes sensitive data from the user.
     *
     * This is important if, at any given point, sensitive information like
     * the plain-text password is stored on this object.
     */
    public function eraseCredentials();

    /**
     * Get username for the user.
     *
     * @return string|null
     */
    public function getUsername(): ?string;

    /**
     * Set username for user.
     *
     * @param string $username
     *
     * @return mixed
     */
    public function setUsername(string $username);

    /**
     * Get canonical username.
     *
     * @return string|null
     */
    public function getUsernameCanonical(): ?string;

    /**
     * Set username canonical.
     *
     * @param string|null $usernameCanonical
     *
     * @return static
     */
    public function setUsernameCanonical(?string $usernameCanonical);

    /**
     * Set email for user.
     *
     * @param string $email
     *
     * @return static
     */
    public function setEmail(string $email);

    /**
     * Get email for user.
     *
     * @return string|null
     */
    public function getEmail(): ?string;

    /**
     * Set canonical email for user.
     *
     * @return string|null
     */
    public function getEmailCanonical(): ?string;

    /**
     * Set canonical email for user.
     *
     * @param string|null $emailCanonical
     *
     * @return static
     */
    public function setEmailCanonical(?string $emailCanonical);

    /**
     * Get salt for user.
     *
     * @return string|null
     */
    public function getSalt(): ?string;

    /**
     * Set salt for user.
     *
     * @param string|null $salt
     *
     * @return static
     */
    public function setSalt(?string $salt);

    /**
     * Get password for the user.
     *
     * @return string|null
     */
    public function getPassword();

    /**
     * Set password for user.
     *
     * @param string $password
     *
     * @return static
     */
    public function setPassword(string $password);

    /**
     * Get plainPassword for user.
     *
     * @return string|null
     */
    public function getPlainPassword(): ?string;

    /**
     * Set plainPassword for user.
     *
     * @param string|null $plainPassword
     *
     * @return static
     */
    public function setPlainPassword(?string $plainPassword);

    /**
     * Get user last login.
     *
     * @return \DateTime|null
     */
    public function getLastLogin(): ?\DateTime;

    /**
     * Set last login for user.
     *
     * @param \DateTime|null $lastLogin
     *
     * @return static
     */
    public function setLastLogin(?\DateTime $lastLogin);

    /**
     * Get user email verification token.
     *
     * @return string|null
     */
    public function getEmailVerificationToken(): ?string;

    /**
     * Set email verification token for user.
     *
     * @param string|null $emailVerificationToken
     *
     * @return static
     */
    public function setEmailVerificationToken(?string $emailVerificationToken);

    /**
     * Get password reset token for user.
     *
     * @return string|null
     */
    public function getPasswordResetToken(): ?string;

    /**
     * Set password reset token for user.
     *
     * @param string|null $passwordResetToken
     *
     * @return static
     */
    public function setPasswordResetToken(?string $passwordResetToken);

    /**
     * Get password requested time.
     *
     * @return \DateTime|null
     */
    public function getPasswordRequestedAt(): ?\DateTime;

    /**
     * Set password requested time.
     *
     * @param \DateTime|null $passwordRequestedAt
     *
     * @return static
     */
    public function setPasswordRequestedAt(?\DateTime $passwordRequestedAt);

    /**
     * Get email verified time.
     *
     * @return \DateTime|null
     */
    public function getEmailVerifiedAt(): ?\DateTime;

    /**
     * Set email verified time.
     *
     * @param \DateTime|null $emailVerifiedAt
     *
     * @return static
     */
    public function setEmailVerifiedAt(?\DateTime $emailVerifiedAt);

    /**
     * Check if user is locked.
     *
     * @return bool
     */
    public function isLocked(): bool;

    /**
     * Set user locked status.
     *
     * @param bool $locked
     *
     * @return static
     */
    public function setLocked(bool $locked);

    /**
     * Get credentials expire time.
     *
     * @return \DateTime|null
     */
    public function getCredentialsExpireAt(): ?\DateTime;

    /**
     * Set credentials expire time.
     *
     * @param \DateTime|null $credentialsExpireAt
     *
     * @return static
     */
    public function setCredentialsExpireAt(?\DateTime $credentialsExpireAt);
}
